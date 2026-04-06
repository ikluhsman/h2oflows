ALTER TABLE reach_access DROP CONSTRAINT reach_access_access_type_check;
ALTER TABLE reach_access ADD CONSTRAINT reach_access_access_type_check
  CHECK (access_type = ANY (ARRAY[
    'put_in', 'take_out', 'shuttle_drop', 'intermediate', 'camp'
  ]));
