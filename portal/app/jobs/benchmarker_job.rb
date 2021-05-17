class BenchmarkerJob < ActiveJob::Base
  queue_as :default

  DEFAULT_TIMEOUT = 100

  def perform(job_id:)
    buf = ''
    pid = nil
    process = nil

    job = Job.find(job_id)
    job.status = 'Running'
    job.save

    timeout = Rails.application.config.x.benchmarker.timeout || DEFAULT_TIMEOUT
    command = Rails.application.config.x.benchmarker.command
    userdata = Rails.application.config.x.benchmarker.userdata

    path = File.dirname(Rails.application.config.x.benchmarker.command)
    args = ['-t', job.team.app_host, '-u', userdata].join(' ')

    logger.info "Command: #{command} #{args}"
    Timeout.timeout(timeout) do
      process = IO.popen("#{command} #{args}")
      pid = process.pid
      while line = process.gets
        buf << line
      end
    end
    logger.info "Result Buffer: #{buf}"
    result = JSON.parse(buf)

    job.team.scores << Score.create(
      pass: result['pass'],
      score: result['score'],
      message: result['messages'].join(' ')
    )
  rescue Timeout::Error => e
    Process.kill('SIGINT', pid) if pid
    process.close if process
    job.team.scores << Score.create(
      pass: 'FAIL',
      score: 0,
      message: 'TIMEOUT'
    )
  rescue => e
    logger.info "Unexpected error: #{e.to_s}"
  ensure
    if job
      job.status = 'Finished'
      job.save
    end
  end
end
