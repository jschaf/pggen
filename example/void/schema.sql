CREATE FUNCTION void_fn() RETURNS void AS
$$
BEGIN
END;
$$ LANGUAGE plpgsql;

-- noinspection SqlUnused
CREATE FUNCTION void_fn_two_params(id int, comment text) RETURNS void AS
$$
BEGIN
END;
$$ LANGUAGE plpgsql;
