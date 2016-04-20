class HomeController < ApplicationController
  before_action :authenticate_user!

  def index
    my_team = current_user.team

    if my_team
      @my_scores = Score.where(team: my_team).order(:created_at).reverse_order
    end

    # Fetch stats current - 60min (5 * 12)
    @chart_data = Score.history(time: Time.new)
    @ordered_score = Score.ordered_stats(time: Time.new, limit: 10)

    @jobs = Job.where(status: ['Running', 'waiting']).order(:id)
  end
end
