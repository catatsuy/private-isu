class BenchmarkerJob < ActiveJob::Base
  queue_as :default

  DEFAULT_TIMEOUT = 60

  def perform(job_id:)
    buf = ''
    pid = nil
    process = nil

    job = Job.find(job_id)
    job.status = 'Running'
    job.save

    timeout = Rails.application.config.x.benchmarker.timeout || DEFAULT_TIMEOUT
    command = Rails.application.config.x.benchmarker.command
    path = File.dirname(Rails.application.config.x.benchmarker.command)
    args = ['-t', job.team.app_host, '-u', "#{path}/userdata"].join(' ')

    Timeout.timeout(timeout) do
      process = IO.popen("#{command} #{args}")
      pid = process.pid
      while line = process.gets
        buf << line
      end
    end
    result = JSON.parse(buf)

    job.team.scores << Score.create(
      pass: result['pass'],
      score: result['score'],
      message: result['message'].join(' ')
    )
  rescue Timeout::Error => e
    Process.kill('SIGINT', pid) if pid
    process.close if process
  ensure
    job.status = 'Finished'
    job.save
  end
end
