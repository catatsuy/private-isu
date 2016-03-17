class HomeController < ApplicationController
  before_action :authenticate_user!

  def index
    @chart_data = []
    @job = Job.new
    jobs_stats = Job.group(:status).count
    @job_stats = {
      running: jobs_stats['Running'] || 0,
      finished: jobs_stats['Finished'] || 0,
      waiting: jobs_stats['waiting'] || 0
    }
  end
end
