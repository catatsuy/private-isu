class Score < ActiveRecord::Base
  belongs_to :team

  # トップNチームのスコア
  def self.ordered_stats(time:, limit:10)
    scores = where('created_at <= ?', time)
    .group('team_id')
    .order('best_score DESC')
    .limit(limit)
    .pluck('team_id', 'MAX(score) AS best_score')

    # Pre-fetch teams (speed)
    team_hash = Team.pluck(:id, :name).to_h

    score_hash = {}
    scores
    .each do |(team_id, best_score)|
      key = team_hash[team_id]
      score_hash[key] = best_score
    end
    score_hash
  end

  def self.history(time:)
    # Pre-fetch teams (speed)
    team_hash = Team.pluck(:id, :name).to_h

    team_scores = Hash.new {|h,key| h[key] = []}

    where('created_at <= ?', time)
    .order('created_at ASC') # インデックス貼ったほうがいいかも
    .each do |score|
      team_scores[team_hash[score.team_id]] << [score.created_at.in_time_zone, score.score]
    end

    # map for graph
    # [
    #   { name: team1.name, data: [['06:00:00', 123], ['06:05:00', 456]]},
    #   { name: team2.name, data: [['06:00:00', 234], ['06:05:00', 345]]},
    #   ...
    # ]
    team_scores.map { |k, v| { name: k, data: v } }
  end
end
