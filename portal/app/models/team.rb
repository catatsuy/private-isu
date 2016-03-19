class Team < ActiveRecord::Base
  has_many :users
  has_many :scores
  has_one :job
end
