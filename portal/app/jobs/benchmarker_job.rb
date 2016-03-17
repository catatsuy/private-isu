class BenchmarkerJob < ActiveJob::Base
  queue_as :default

  DEFAULT_TIMEOUT = 60

  def perform(job_id:)
    job = Job.find(job_id)
    job.status = 'Running'
    job.save

    timeout = Rails.application.config.x.benchmarker.timeout || DEFAULT_TIMEOUT
    command = Rails.application.config.x.benchmarker.command

    Timeout.timeout(timeout) do
      process = IO.popen(command)
      pid = process.pid
      buf = ''
      while line = process.gets
        buf << line
      end
    end
  rescue Timeout::Error => e
    Process.kill('SIGINT', pid)
    process.close if process
  ensure
    job.status = 'Finished'
    job.save
  end
end
