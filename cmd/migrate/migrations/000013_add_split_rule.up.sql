ALTER TABLE expense
    ADD COLUMN split_rule TEXT DEFAULT 'Equally' NOT NULL CHECK (split_rule IN ('Equally', 'Unequally', 'You-Half', 'You-Full', 'Other-Half', 'Other-Full'));