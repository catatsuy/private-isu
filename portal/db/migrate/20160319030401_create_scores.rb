class CreateScores < ActiveRecord::Migration
  def change
    create_table :scores do |t|
      t.integer :score
      t.references :team
      t.timestamps null: false
    end
  end
end
