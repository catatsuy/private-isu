class JobsController < ApplicationController
  before_action :authenticate_user!

  def create
    if Job.time_wait?(team: current_user.team)
      flash[:alert] = 'Job already enqueued. Please wait and try later.'
    else
      job = Job.create(team: current_user.team, status: 'Waiting')
      BenchmarkerJob.perform_later(job_id: job.id)
    end
  ensure
    redirect_to :root
  end
end
