class Team < ActiveRecord::Base
  has_many :users
  has_one :job
end
