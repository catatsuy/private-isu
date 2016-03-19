class HomeController < ApplicationController
  before_action :authenticate_user!

  def index
    # Fetch stats current - 60min (5 * 12)
    @chart_data = Score.stats(time: Time.new, slice: 5, limit: 12)
    @ordered_score = Score.ordered_stats(time: Time.new)

    @job = Job.new
    jobs_stats = Job.group(:status).count
    @job_stats = {
      running: jobs_stats['Running'] || 0,
      finished: jobs_stats['Finished'] || 0,
      waiting: jobs_stats['waiting'] || 0
    }
  end
end
