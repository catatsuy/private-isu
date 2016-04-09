class HomeController < ApplicationController
  before_action :authenticate_user!

  def index
    my_team = current_user.team

    if my_team
      @my_scores = Score.where(team: my_team).order(:created_at).reverse_order
    end

    # Fetch stats current - 60min (5 * 12)
    @chart_data = Score.stats(time: Time.new, slice: 5, limit: 12)
    @ordered_score = Score.ordered_stats(time: Time.new, limit: 10)

    @job = Job.new
    jobs_stats = Job.group(:status).count
    @job_stats = {
      running: jobs_stats['Running'] || 0,
      finished: jobs_stats['Finished'] || 0,
      waiting: jobs_stats['waiting'] || 0
    }
  end
end
