class Score < ActiveRecord::Base
  belongs_to :team

  # Generate score hash for graph (dirty)
  def self.stats(time:, slice:, limit:)
    # time round per :slice
    rounded_t = Time.local(time.year, time.month, time.day, time.hour, time.min/slice*slice)
    # Pre-fetch teams (speed)
    team_hash = Team.pluck(:id, :name).to_h

    score_hash = {}
    # Fetch latest score per :slice
    # SELECT "scores".* FROM "scores" WHERE (created_at < '2016-03-19 06:00:00') GROUP BY "scores"."team_id"  ORDER BY "scores"."created_at" DESC
    limit.times do |n|
      t = rounded_t - (n*slice).minutes
      scores = where('created_at < ?', t).order(:created_at).reverse_order.group(:team_id)
      scores.each do |score|
        key = team_hash[score.team_id]
        score_hash[key] = [] unless score_hash[key]
        score_hash[key] << [t, score.score]
      end
    end

    # map for graph
    # [
    #   { name: team1.name, data: [['06:00:00', 123], ['06:05:00', 456]]},
    #   { name: team2.name, data: [['06:00:00', 234], ['06:05:00', 345]]},
    #   ...
    # ]
    score_hash.map { |k, v| { name: k, data: v } }
  end
end
