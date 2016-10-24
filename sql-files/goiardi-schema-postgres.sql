--
-- PostgreSQL database dump
--

-- Dumped from database version 9.5.3
-- Dumped by pg_dump version 9.5.3

SET statement_timeout = 0;
SET lock_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: goiardi; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA goiardi;


--
-- Name: sqitch; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA sqitch;


--
-- Name: SCHEMA sqitch; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON SCHEMA sqitch IS 'Sqitch database deployment metadata v1.0.';


--
-- Name: plpgsql; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;


--
-- Name: EXTENSION plpgsql; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION plpgsql IS 'PL/pgSQL procedural language';


--
-- Name: ltree; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS ltree WITH SCHEMA goiardi;


--
-- Name: EXTENSION ltree; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION ltree IS 'data type for hierarchical tree-like structures';


--
-- Name: pg_trgm; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA goiardi;


--
-- Name: EXTENSION pg_trgm; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION pg_trgm IS 'text similarity measurement and index searching based on trigrams';


SET search_path = goiardi, pg_catalog;

--
-- Name: log_action; Type: TYPE; Schema: goiardi; Owner: -
--

CREATE TYPE log_action AS ENUM (
    'create',
    'delete',
    'modify'
);


--
-- Name: log_actor; Type: TYPE; Schema: goiardi; Owner: -
--

CREATE TYPE log_actor AS ENUM (
    'user',
    'client'
);


--
-- Name: report_status; Type: TYPE; Schema: goiardi; Owner: -
--

CREATE TYPE report_status AS ENUM (
    'started',
    'success',
    'failure'
);


--
-- Name: shovey_output; Type: TYPE; Schema: goiardi; Owner: -
--

CREATE TYPE shovey_output AS ENUM (
    'stdout',
    'stderr'
);


--
-- Name: status_node; Type: TYPE; Schema: goiardi; Owner: -
--

CREATE TYPE status_node AS ENUM (
    'new',
    'up',
    'down'
);


--
-- Name: delete_search_collection(text, integer); Type: FUNCTION; Schema: goiardi; Owner: -
--

CREATE FUNCTION delete_search_collection(col text, m_organization_id integer) RETURNS void
    LANGUAGE plpgsql
    AS $$
DECLARE
	sc_id bigint;
BEGIN
	SELECT id INTO sc_id FROM goiardi.search_collections WHERE name = col AND organization_id = m_organization_id;
	IF NOT FOUND THEN
		RAISE EXCEPTION 'The collection % does not exist!', col;
	END IF;
	DELETE FROM goiardi.search_items WHERE organization_id = m_organization_id AND search_collection_id = sc_id;
	DELETE FROM goiardi.search_collections WHERE organization_id = m_organization_id AND id = sc_id;
END;
$$;


--
-- Name: delete_search_item(text, text, integer); Type: FUNCTION; Schema: goiardi; Owner: -
--

CREATE FUNCTION delete_search_item(col text, item text, m_organization_id integer) RETURNS void
    LANGUAGE plpgsql
    AS $$
DECLARE
	sc_id bigint;
BEGIN
	SELECT id INTO sc_id FROM goiardi.search_collections WHERE name = col AND organization_id = m_organization_id;
	IF NOT FOUND THEN
		RAISE EXCEPTION 'The collection % does not exist!', col;
	END IF;
	DELETE FROM goiardi.search_items WHERE organization_id = m_organization_id AND search_collection_id = sc_id AND item_name = item;
END;
$$;


--
-- Name: insert_dbi(text, text, text, bigint, jsonb); Type: FUNCTION; Schema: goiardi; Owner: -
--

CREATE FUNCTION insert_dbi(m_data_bag_name text, m_name text, m_orig_name text, m_dbag_id bigint, m_raw_data jsonb) RETURNS bigint
    LANGUAGE plpgsql
    AS $$
DECLARE
	u BIGINT;
	dbi_id BIGINT;
BEGIN
	SELECT id INTO u FROM goiardi.data_bags WHERE id = m_dbag_id;
	IF NOT FOUND THEN
		RAISE EXCEPTION 'aiiiie! The data bag % was deleted from the db while we were doing something else', m_data_bag_name;
	END IF;

	INSERT INTO goiardi.data_bag_items (name, orig_name, data_bag_id, raw_data, created_at, updated_at) VALUES (m_name, m_orig_name, m_dbag_id, m_raw_data, NOW(), NOW()) RETURNING id INTO dbi_id;
	RETURN dbi_id;
END;
$$;


--
-- Name: insert_node_status(text, status_node); Type: FUNCTION; Schema: goiardi; Owner: -
--

CREATE FUNCTION insert_node_status(m_name text, m_status status_node) RETURNS void
    LANGUAGE plpgsql
    AS $$
DECLARE
	n BIGINT;
BEGIN
	SELECT id INTO n FROM goiardi.nodes WHERE name = m_name;
	IF NOT FOUND THEN
		RAISE EXCEPTION 'aiiie, the node % was deleted while we were doing something else trying to insert a status', m_name;
	END IF;
	INSERT INTO goiardi.node_statuses (node_id, status, updated_at) VALUES (n, m_status, NOW());
	RETURN;
END;
$$;


--
-- Name: merge_clients(text, text, boolean, boolean, text, text); Type: FUNCTION; Schema: goiardi; Owner: -
--

CREATE FUNCTION merge_clients(m_name text, m_nodename text, m_validator boolean, m_admin boolean, m_public_key text, m_certificate text) RETURNS void
    LANGUAGE plpgsql
    AS $$
DECLARE
    u_id bigint;
    u_name text;
BEGIN
    SELECT id, name INTO u_id, u_name FROM goiardi.users WHERE name = m_name;
    IF FOUND THEN
        RAISE EXCEPTION 'a user with id % named % was found that would conflict with this client', u_id, u_name;
    END IF;
    LOOP
        -- first try to update the key
        UPDATE goiardi.clients SET name = m_name, nodename = m_nodename, validator = m_validator, admin = m_admin, public_key = m_public_key, certificate = m_certificate, updated_at = NOW() WHERE name = m_name;
        IF found THEN
            RETURN;
        END IF;
        -- not there, so try to insert the key
        -- if someone else inserts the same key concurrently,
        -- we could get a unique-key failure
        BEGIN
            INSERT INTO goiardi.clients (name, nodename, validator, admin, public_key, certificate, created_at, updated_at) VALUES (m_name, m_nodename, m_validator, m_admin, m_public_key, m_certificate, NOW(), NOW());
            RETURN;
        EXCEPTION WHEN unique_violation THEN
            -- Do nothing, and loop to try the UPDATE again.
        END;
    END LOOP;
END;
$$;


--
-- Name: merge_cookbook_versions(bigint, boolean, jsonb, jsonb, jsonb, jsonb, jsonb, jsonb, jsonb, jsonb, jsonb, jsonb, bigint, bigint, bigint); Type: FUNCTION; Schema: goiardi; Owner: -
--

CREATE FUNCTION merge_cookbook_versions(c_id bigint, is_frozen boolean, defb jsonb, libb jsonb, attb jsonb, recb jsonb, prob jsonb, resb jsonb, temb jsonb, roob jsonb, filb jsonb, metb jsonb, maj bigint, min bigint, patch bigint) RETURNS bigint
    LANGUAGE plpgsql
    AS $$
DECLARE
    cbv_id BIGINT;
BEGIN
    LOOP
        -- first try to update the key
        UPDATE goiardi.cookbook_versions SET frozen = is_frozen, metadata = metb, definitions = defb, libraries = libb, attributes = attb, recipes = recb, providers = prob, resources = resb, templates = temb, root_files = roob, files = filb, updated_at = NOW() WHERE cookbook_id = c_id AND major_ver = maj AND minor_ver = min AND patch_ver = patch RETURNING id INTO cbv_id;
        IF found THEN
            RETURN cbv_id;
        END IF;
        -- not there, so try to insert the key
        -- if someone else inserts the same key concurrently,
        -- we could get a unique-key failure
        BEGIN
            INSERT INTO goiardi.cookbook_versions (cookbook_id, major_ver, minor_ver, patch_ver, frozen, metadata, definitions, libraries, attributes, recipes, providers, resources, templates, root_files, files, created_at, updated_at) VALUES (c_id, maj, min, patch, is_frozen, metb, defb, libb, attb, recb, prob, resb, temb, roob, filb, NOW(), NOW()) RETURNING id INTO cbv_id;
            RETURN c_id;
        EXCEPTION WHEN unique_violation THEN
            -- Do nothing, and loop to try the UPDATE again.
        END;
    END LOOP;
END;
$$;


--
-- Name: merge_cookbooks(text); Type: FUNCTION; Schema: goiardi; Owner: -
--

CREATE FUNCTION merge_cookbooks(m_name text) RETURNS bigint
    LANGUAGE plpgsql
    AS $$
DECLARE
    c_id BIGINT;
BEGIN
    LOOP
        -- first try to update the key
        UPDATE goiardi.cookbooks SET name = m_name, updated_at = NOW() WHERE name = m_name RETURNING id INTO c_id;
        IF found THEN
            RETURN c_id;
        END IF;
        -- not there, so try to insert the key
        -- if someone else inserts the same key concurrently,
        -- we could get a unique-key failure
        BEGIN
            INSERT INTO goiardi.cookbooks (name, created_at, updated_at) VALUES (m_name, NOW(), NOW()) RETURNING id INTO c_id;
            RETURN c_id;
        EXCEPTION WHEN unique_violation THEN
            -- Do nothing, and loop to try the UPDATE again.
        END;
    END LOOP;
END;
$$;


--
-- Name: merge_data_bags(text); Type: FUNCTION; Schema: goiardi; Owner: -
--

CREATE FUNCTION merge_data_bags(m_name text) RETURNS bigint
    LANGUAGE plpgsql
    AS $$
DECLARE
    db_id BIGINT;
BEGIN
    LOOP
        -- first try to update the key
        UPDATE goiardi.data_bags SET updated_at = NOW() WHERE name = m_name RETURNING id INTO db_id;
        IF found THEN
            RETURN db_id;
        END IF;
        -- not there, so try to insert the key
        -- if someone else inserts the same key concurrently,
        -- we could get a unique-key failure
        BEGIN
            INSERT INTO goiardi.data_bags (name, created_at, updated_at) VALUES (m_name, NOW(), NOW()) RETURNING id INTO db_id;
            RETURN db_id;
        EXCEPTION WHEN unique_violation THEN
            -- Do nothing, and loop to try the UPDATE again.
        END;
    END LOOP;
END;
$$;


--
-- Name: merge_environments(text, text, jsonb, jsonb, jsonb); Type: FUNCTION; Schema: goiardi; Owner: -
--

CREATE FUNCTION merge_environments(m_name text, m_description text, m_default_attr jsonb, m_override_attr jsonb, m_cookbook_vers jsonb) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    LOOP
        -- first try to update the key
	UPDATE goiardi.environments SET description = m_description, default_attr = m_default_attr, override_attr = m_override_attr, cookbook_vers = m_cookbook_vers, updated_at = NOW() WHERE name = m_name;
	IF found THEN
		RETURN;
	END IF;
        -- not there, so try to insert the key
        -- if someone else inserts the same key concurrently,
        -- we could get a unique-key failure
        BEGIN
	    INSERT INTO goiardi.environments (name, description, default_attr, override_attr, cookbook_vers, created_at, updated_at) VALUES (m_name, m_description, m_default_attr, m_override_attr, m_cookbook_vers, NOW(), NOW());
            RETURN;
        EXCEPTION WHEN unique_violation THEN
            -- Do nothing, and loop to try the UPDATE again.
        END;
    END LOOP;
END;
$$;


--
-- Name: merge_nodes(text, text, jsonb, jsonb, jsonb, jsonb, jsonb); Type: FUNCTION; Schema: goiardi; Owner: -
--

CREATE FUNCTION merge_nodes(m_name text, m_chef_environment text, m_run_list jsonb, m_automatic_attr jsonb, m_normal_attr jsonb, m_default_attr jsonb, m_override_attr jsonb) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    LOOP
        -- first try to update the key
	UPDATE goiardi.nodes SET chef_environment = m_chef_environment, run_list = m_run_list, automatic_attr = m_automatic_attr, normal_attr = m_normal_attr, default_attr = m_default_attr, override_attr = m_override_attr, updated_at = NOW() WHERE name = m_name;
	IF found THEN
	    RETURN;
	END IF;
        -- not there, so try to insert the key
        -- if someone else inserts the same key concurrently,
        -- we could get a unique-key failure
        BEGIN
	    INSERT INTO goiardi.nodes (name, chef_environment, run_list, automatic_attr, normal_attr, default_attr, override_attr, created_at, updated_at) VALUES (m_name, m_chef_environment, m_run_list, m_automatic_attr, m_normal_attr, m_default_attr, m_override_attr, NOW(), NOW());
            RETURN;
        EXCEPTION WHEN unique_violation THEN
            -- Do nothing, and loop to try the UPDATE again.
        END;
    END LOOP;
END;
$$;


--
-- Name: merge_reports(uuid, text, timestamp with time zone, timestamp with time zone, integer, report_status, text, jsonb, jsonb); Type: FUNCTION; Schema: goiardi; Owner: -
--

CREATE FUNCTION merge_reports(m_run_id uuid, m_node_name text, m_start_time timestamp with time zone, m_end_time timestamp with time zone, m_total_res_count integer, m_status report_status, m_run_list text, m_resources jsonb, m_data jsonb) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    LOOP
        -- first try to update the key
	UPDATE goiardi.reports SET start_time = m_start_time, end_time = m_end_time, total_res_count = m_total_res_count, status = m_status, run_list = m_run_list, resources = m_resources, data = m_data, updated_at = NOW() WHERE run_id = m_run_id;
	IF found THEN
	    RETURN;
	END IF;
        -- not there, so try to insert the key
        -- if someone else inserts the same key concurrently,
        -- we could get a unique-key failure
        BEGIN
	    INSERT INTO goiardi.reports (run_id, node_name, start_time, end_time, total_res_count, status, run_list, resources, data, created_at, updated_at) VALUES (m_run_id, m_node_name, m_start_time, m_end_time, m_total_res_count, m_status, m_run_list, m_resources, m_data, NOW(), NOW());
            RETURN;
        EXCEPTION WHEN unique_violation THEN
            -- Do nothing, and loop to try the UPDATE again.
        END;
    END LOOP;
END;
$$;


--
-- Name: merge_roles(text, text, jsonb, jsonb, jsonb, jsonb); Type: FUNCTION; Schema: goiardi; Owner: -
--

CREATE FUNCTION merge_roles(m_name text, m_description text, m_run_list jsonb, m_env_run_lists jsonb, m_default_attr jsonb, m_override_attr jsonb) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    LOOP
        -- first try to update the key
	UPDATE goiardi.roles SET description = m_description, run_list = m_run_list, env_run_lists = m_env_run_lists, default_attr = m_default_attr, override_attr = m_override_attr, updated_at = NOW() WHERE name = m_name;
	IF found THEN
	    RETURN;
	END IF;
        -- not there, so try to insert the key
        -- if someone else inserts the same key concurrently,
        -- we could get a unique-key failure
        BEGIN
	    INSERT INTO goiardi.roles (name, description, run_list, env_run_lists, default_attr, override_attr, created_at, updated_at) VALUES (m_name, m_description, m_run_list, m_env_run_lists, m_default_attr, m_override_attr, NOW(), NOW());
            RETURN;
        EXCEPTION WHEN unique_violation THEN
            -- Do nothing, and loop to try the UPDATE again.
        END;
    END LOOP;
END;
$$;


--
-- Name: merge_sandboxes(character varying, timestamp with time zone, jsonb, boolean); Type: FUNCTION; Schema: goiardi; Owner: -
--

CREATE FUNCTION merge_sandboxes(m_sbox_id character varying, m_creation_time timestamp with time zone, m_checksums jsonb, m_completed boolean) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    LOOP
        -- first try to update the key
	UPDATE goiardi.sandboxes SET checksums = m_checksums, completed = m_completed WHERE sbox_id = m_sbox_id;
	IF found THEN
	    RETURN;
	END IF;
        -- not there, so try to insert the key
        -- if someone else inserts the same key concurrently,
        -- we could get a unique-key failure
        BEGIN
	    INSERT INTO goiardi.sandboxes (sbox_id, creation_time, checksums, completed) VALUES (m_sbox_id, m_creation_time, m_checksums, m_completed);
            RETURN;
        EXCEPTION WHEN unique_violation THEN
            -- Do nothing, and loop to try the UPDATE again.
        END;
    END LOOP;
END;
$$;


--
-- Name: merge_shovey_runs(uuid, text, text, timestamp with time zone, timestamp with time zone, text, integer); Type: FUNCTION; Schema: goiardi; Owner: -
--

CREATE FUNCTION merge_shovey_runs(m_shovey_run_id uuid, m_node_name text, m_status text, m_ack_time timestamp with time zone, m_end_time timestamp with time zone, m_error text, m_exit_status integer) RETURNS void
    LANGUAGE plpgsql
    AS $$
DECLARE
    m_shovey_id bigint;
BEGIN
    LOOP
	UPDATE goiardi.shovey_runs SET status = m_status, ack_time = NULLIF(m_ack_time, '0001-01-01 00:00:00 +0000'), end_time = NULLIF(m_end_time, '0001-01-01 00:00:00 +0000'), error = m_error, exit_status = cast(m_exit_status as smallint) WHERE shovey_uuid = m_shovey_run_id AND node_name = m_node_name;
	IF found THEN
	    RETURN;
	END IF;
	BEGIN
	    SELECT id INTO m_shovey_id FROM goiardi.shoveys WHERE run_id = m_shovey_run_id;
	    INSERT INTO goiardi.shovey_runs (shovey_uuid, shovey_id, node_name, status, ack_time, end_time, error, exit_status) VALUES (m_shovey_run_id, m_shovey_id, m_node_name, m_status, NULLIF(m_ack_time, '0001-01-01 00:00:00 +0000'),NULLIF(m_end_time, '0001-01-01 00:00:00 +0000'), m_error, cast(m_exit_status as smallint));
	EXCEPTION WHEN unique_violation THEN
	    -- meh.
	END;
    END LOOP;
END;
$$;


--
-- Name: merge_shoveys(uuid, text, text, bigint, character varying); Type: FUNCTION; Schema: goiardi; Owner: -
--

CREATE FUNCTION merge_shoveys(m_run_id uuid, m_command text, m_status text, m_timeout bigint, m_quorum character varying) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    LOOP
	UPDATE goiardi.shoveys SET status = m_status, updated_at = NOW() WHERE run_id = m_run_id;
        IF found THEN
	    RETURN;
    	END IF;
    	BEGIN
	    INSERT INTO goiardi.shoveys (run_id, command, status, timeout, quorum, created_at, updated_at) VALUES (m_run_id, m_command, m_status, m_timeout, m_quorum, NOW(), NOW());
            RETURN;
        EXCEPTION WHEN unique_violation THEN
            -- moo.
    	END;
    END LOOP;
END;
$$;


--
-- Name: merge_users(text, text, text, boolean, text, character varying, bytea, bigint); Type: FUNCTION; Schema: goiardi; Owner: -
--

CREATE FUNCTION merge_users(m_name text, m_displayname text, m_email text, m_admin boolean, m_public_key text, m_passwd character varying, m_salt bytea, m_organization_id bigint) RETURNS void
    LANGUAGE plpgsql
    AS $$
DECLARE
    c_id bigint;
    c_name text;
BEGIN
    SELECT id, name INTO c_id, c_name FROM goiardi.clients WHERE name = m_name AND organization_id = m_organization_id;
    IF FOUND THEN
        RAISE EXCEPTION 'a client with id % named % was found that would conflict with this client', c_id, c_name;
    END IF;
    IF m_email = '' THEN
        m_email := NULL;
    END IF;
    LOOP
        -- first try to update the key
        UPDATE goiardi.users SET name = m_name, displayname = m_displayname, email = m_email, admin = m_admin, public_key = m_public_key, passwd = m_passwd, salt = m_salt, updated_at = NOW() WHERE name = m_name;
        IF found THEN
            RETURN;
        END IF;
        -- not there, so try to insert the key
        -- if someone else inserts the same key concurrently,
        -- we could get a unique-key failure
        BEGIN
            INSERT INTO goiardi.users (name, displayname, email, admin, public_key, passwd, salt, created_at, updated_at) VALUES (m_name, m_displayname, m_email, m_admin, m_public_key, m_passwd, m_salt, NOW(), NOW());
            RETURN;
        END;
    END LOOP;
END;
$$;


--
-- Name: rename_client(text, text); Type: FUNCTION; Schema: goiardi; Owner: -
--

CREATE FUNCTION rename_client(old_name text, new_name text) RETURNS void
    LANGUAGE plpgsql
    AS $$
DECLARE
	u_id bigint;
	u_name text;
BEGIN
	SELECT id, name INTO u_id, u_name FROM goiardi.users WHERE name = new_name;
	IF FOUND THEN
		RAISE EXCEPTION 'a user with id % named % was found that would conflict with this client', u_id, u_name;
	END IF;
	BEGIN
		UPDATE goiardi.clients SET name = new_name WHERE name = old_name;
	EXCEPTION WHEN unique_violation THEN
		RAISE EXCEPTION 'Client % already exists, cannot rename %', old_name, new_name;
	END;
END;
$$;


--
-- Name: rename_user(text, text, integer); Type: FUNCTION; Schema: goiardi; Owner: -
--

CREATE FUNCTION rename_user(old_name text, new_name text, m_organization_id integer) RETURNS void
    LANGUAGE plpgsql
    AS $$
DECLARE
	c_id bigint;
	c_name text;
BEGIN
	SELECT id, name INTO c_id, c_name FROM goiardi.clients WHERE name = new_name AND organization_id = m_organization_id;
	IF FOUND THEN
		RAISE EXCEPTION 'a client with id % named % was found that would conflict with this user', c_id, c_name;
	END IF;
	BEGIN
		UPDATE goiardi.users SET name = new_name WHERE name = old_name;
	EXCEPTION WHEN unique_violation THEN
		RAISE EXCEPTION 'User % already exists, cannot rename %', old_name, new_name;
	END;
END;
$$;


SET default_tablespace = '';

SET default_with_oids = false;

--
-- Name: clients; Type: TABLE; Schema: goiardi; Owner: -
--

CREATE TABLE clients (
    id bigint NOT NULL,
    name text NOT NULL,
    nodename text,
    validator boolean,
    admin boolean,
    organization_id bigint DEFAULT 1 NOT NULL,
    public_key text,
    certificate text,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: clients_id_seq; Type: SEQUENCE; Schema: goiardi; Owner: -
--

CREATE SEQUENCE clients_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: clients_id_seq; Type: SEQUENCE OWNED BY; Schema: goiardi; Owner: -
--

ALTER SEQUENCE clients_id_seq OWNED BY clients.id;


--
-- Name: cookbook_versions; Type: TABLE; Schema: goiardi; Owner: -
--

CREATE TABLE cookbook_versions (
    id bigint NOT NULL,
    cookbook_id bigint NOT NULL,
    major_ver bigint NOT NULL,
    minor_ver bigint NOT NULL,
    patch_ver bigint DEFAULT 0 NOT NULL,
    frozen boolean,
    metadata jsonb,
    definitions jsonb,
    libraries jsonb,
    attributes jsonb,
    recipes jsonb,
    providers jsonb,
    resources jsonb,
    templates jsonb,
    root_files jsonb,
    files jsonb,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: cookbook_versions_id_seq; Type: SEQUENCE; Schema: goiardi; Owner: -
--

CREATE SEQUENCE cookbook_versions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: cookbook_versions_id_seq; Type: SEQUENCE OWNED BY; Schema: goiardi; Owner: -
--

ALTER SEQUENCE cookbook_versions_id_seq OWNED BY cookbook_versions.id;


--
-- Name: cookbooks; Type: TABLE; Schema: goiardi; Owner: -
--

CREATE TABLE cookbooks (
    id bigint NOT NULL,
    name text NOT NULL,
    organization_id bigint DEFAULT 1 NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: cookbooks_id_seq; Type: SEQUENCE; Schema: goiardi; Owner: -
--

CREATE SEQUENCE cookbooks_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: cookbooks_id_seq; Type: SEQUENCE OWNED BY; Schema: goiardi; Owner: -
--

ALTER SEQUENCE cookbooks_id_seq OWNED BY cookbooks.id;


--
-- Name: data_bag_items; Type: TABLE; Schema: goiardi; Owner: -
--

CREATE TABLE data_bag_items (
    id bigint NOT NULL,
    name text NOT NULL,
    orig_name text NOT NULL,
    data_bag_id bigint NOT NULL,
    raw_data jsonb,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: data_bag_items_id_seq; Type: SEQUENCE; Schema: goiardi; Owner: -
--

CREATE SEQUENCE data_bag_items_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: data_bag_items_id_seq; Type: SEQUENCE OWNED BY; Schema: goiardi; Owner: -
--

ALTER SEQUENCE data_bag_items_id_seq OWNED BY data_bag_items.id;


--
-- Name: data_bags; Type: TABLE; Schema: goiardi; Owner: -
--

CREATE TABLE data_bags (
    id bigint NOT NULL,
    name text NOT NULL,
    organization_id bigint DEFAULT 1 NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: data_bags_id_seq; Type: SEQUENCE; Schema: goiardi; Owner: -
--

CREATE SEQUENCE data_bags_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: data_bags_id_seq; Type: SEQUENCE OWNED BY; Schema: goiardi; Owner: -
--

ALTER SEQUENCE data_bags_id_seq OWNED BY data_bags.id;


--
-- Name: environments; Type: TABLE; Schema: goiardi; Owner: -
--

CREATE TABLE environments (
    id bigint NOT NULL,
    name text,
    organization_id bigint DEFAULT 1 NOT NULL,
    description text,
    default_attr jsonb,
    override_attr jsonb,
    cookbook_vers jsonb,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: environments_id_seq; Type: SEQUENCE; Schema: goiardi; Owner: -
--

CREATE SEQUENCE environments_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: environments_id_seq; Type: SEQUENCE OWNED BY; Schema: goiardi; Owner: -
--

ALTER SEQUENCE environments_id_seq OWNED BY environments.id;


--
-- Name: file_checksums; Type: TABLE; Schema: goiardi; Owner: -
--

CREATE TABLE file_checksums (
    id bigint NOT NULL,
    organization_id bigint DEFAULT 1 NOT NULL,
    checksum character varying(32)
);


--
-- Name: file_checksums_id_seq; Type: SEQUENCE; Schema: goiardi; Owner: -
--

CREATE SEQUENCE file_checksums_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: file_checksums_id_seq; Type: SEQUENCE OWNED BY; Schema: goiardi; Owner: -
--

ALTER SEQUENCE file_checksums_id_seq OWNED BY file_checksums.id;


--
-- Name: joined_cookbook_version; Type: VIEW; Schema: goiardi; Owner: -
--

CREATE VIEW joined_cookbook_version AS
 SELECT v.major_ver,
    v.minor_ver,
    v.patch_ver,
    ((((v.major_ver || '.'::text) || v.minor_ver) || '.'::text) || v.patch_ver) AS version,
    v.id,
    v.metadata,
    v.recipes,
    c.organization_id,
    c.name
   FROM (cookbooks c
     JOIN cookbook_versions v ON ((c.id = v.cookbook_id)));


--
-- Name: log_infos; Type: TABLE; Schema: goiardi; Owner: -
--

CREATE TABLE log_infos (
    id bigint NOT NULL,
    actor_id bigint DEFAULT 0 NOT NULL,
    actor_info text,
    actor_type log_actor NOT NULL,
    organization_id bigint DEFAULT '1'::bigint NOT NULL,
    "time" timestamp with time zone DEFAULT now(),
    action log_action NOT NULL,
    object_type text NOT NULL,
    object_name text NOT NULL,
    extended_info text
);
ALTER TABLE ONLY log_infos ALTER COLUMN extended_info SET STORAGE EXTERNAL;


--
-- Name: log_infos_id_seq; Type: SEQUENCE; Schema: goiardi; Owner: -
--

CREATE SEQUENCE log_infos_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: log_infos_id_seq; Type: SEQUENCE OWNED BY; Schema: goiardi; Owner: -
--

ALTER SEQUENCE log_infos_id_seq OWNED BY log_infos.id;


--
-- Name: node_statuses; Type: TABLE; Schema: goiardi; Owner: -
--

CREATE TABLE node_statuses (
    id bigint NOT NULL,
    node_id bigint NOT NULL,
    status status_node DEFAULT 'new'::status_node NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: nodes; Type: TABLE; Schema: goiardi; Owner: -
--

CREATE TABLE nodes (
    id bigint NOT NULL,
    name text NOT NULL,
    organization_id bigint DEFAULT 1 NOT NULL,
    chef_environment text DEFAULT '_default'::text NOT NULL,
    run_list jsonb,
    automatic_attr jsonb,
    normal_attr jsonb,
    default_attr jsonb,
    override_attr jsonb,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    is_down boolean DEFAULT false
);


--
-- Name: node_latest_statuses; Type: VIEW; Schema: goiardi; Owner: -
--

CREATE VIEW node_latest_statuses AS
 SELECT DISTINCT ON (n.id) n.id,
    n.name,
    n.chef_environment,
    n.run_list,
    n.automatic_attr,
    n.normal_attr,
    n.default_attr,
    n.override_attr,
    n.is_down,
    ns.status,
    ns.updated_at
   FROM (nodes n
     JOIN node_statuses ns ON ((n.id = ns.node_id)))
  ORDER BY n.id, ns.updated_at DESC;


--
-- Name: node_statuses_id_seq; Type: SEQUENCE; Schema: goiardi; Owner: -
--

CREATE SEQUENCE node_statuses_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: node_statuses_id_seq; Type: SEQUENCE OWNED BY; Schema: goiardi; Owner: -
--

ALTER SEQUENCE node_statuses_id_seq OWNED BY node_statuses.id;


--
-- Name: nodes_id_seq; Type: SEQUENCE; Schema: goiardi; Owner: -
--

CREATE SEQUENCE nodes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: nodes_id_seq; Type: SEQUENCE OWNED BY; Schema: goiardi; Owner: -
--

ALTER SEQUENCE nodes_id_seq OWNED BY nodes.id;


--
-- Name: organizations; Type: TABLE; Schema: goiardi; Owner: -
--

CREATE TABLE organizations (
    id bigint NOT NULL,
    name text NOT NULL,
    description text,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: organizations_id_seq; Type: SEQUENCE; Schema: goiardi; Owner: -
--

CREATE SEQUENCE organizations_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: organizations_id_seq; Type: SEQUENCE OWNED BY; Schema: goiardi; Owner: -
--

ALTER SEQUENCE organizations_id_seq OWNED BY organizations.id;


--
-- Name: reports; Type: TABLE; Schema: goiardi; Owner: -
--

CREATE TABLE reports (
    id bigint NOT NULL,
    run_id uuid NOT NULL,
    node_name character varying(255),
    organization_id bigint DEFAULT 1 NOT NULL,
    start_time timestamp with time zone,
    end_time timestamp with time zone,
    total_res_count integer DEFAULT 0,
    status report_status,
    run_list text,
    resources jsonb,
    data jsonb,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);
ALTER TABLE ONLY reports ALTER COLUMN run_list SET STORAGE EXTERNAL;


--
-- Name: reports_id_seq; Type: SEQUENCE; Schema: goiardi; Owner: -
--

CREATE SEQUENCE reports_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: reports_id_seq; Type: SEQUENCE OWNED BY; Schema: goiardi; Owner: -
--

ALTER SEQUENCE reports_id_seq OWNED BY reports.id;


--
-- Name: roles; Type: TABLE; Schema: goiardi; Owner: -
--

CREATE TABLE roles (
    id bigint NOT NULL,
    name text NOT NULL,
    organization_id bigint DEFAULT 1 NOT NULL,
    description text,
    run_list jsonb,
    env_run_lists jsonb,
    default_attr jsonb,
    override_attr jsonb,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: roles_id_seq; Type: SEQUENCE; Schema: goiardi; Owner: -
--

CREATE SEQUENCE roles_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: roles_id_seq; Type: SEQUENCE OWNED BY; Schema: goiardi; Owner: -
--

ALTER SEQUENCE roles_id_seq OWNED BY roles.id;


--
-- Name: sandboxes; Type: TABLE; Schema: goiardi; Owner: -
--

CREATE TABLE sandboxes (
    id bigint NOT NULL,
    sbox_id character varying(32) NOT NULL,
    organization_id bigint DEFAULT 1 NOT NULL,
    creation_time timestamp with time zone NOT NULL,
    checksums jsonb,
    completed boolean
);


--
-- Name: sandboxes_id_seq; Type: SEQUENCE; Schema: goiardi; Owner: -
--

CREATE SEQUENCE sandboxes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: sandboxes_id_seq; Type: SEQUENCE OWNED BY; Schema: goiardi; Owner: -
--

ALTER SEQUENCE sandboxes_id_seq OWNED BY sandboxes.id;


--
-- Name: search_collections; Type: TABLE; Schema: goiardi; Owner: -
--

CREATE TABLE search_collections (
    id bigint NOT NULL,
    organization_id bigint DEFAULT 1 NOT NULL,
    name text
);


--
-- Name: search_collections_id_seq; Type: SEQUENCE; Schema: goiardi; Owner: -
--

CREATE SEQUENCE search_collections_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: search_collections_id_seq; Type: SEQUENCE OWNED BY; Schema: goiardi; Owner: -
--

ALTER SEQUENCE search_collections_id_seq OWNED BY search_collections.id;


--
-- Name: search_items; Type: TABLE; Schema: goiardi; Owner: -
--

CREATE TABLE search_items (
    id bigint NOT NULL,
    organization_id bigint DEFAULT 1 NOT NULL,
    search_collection_id bigint NOT NULL,
    item_name text,
    value text,
    path ltree
);


--
-- Name: search_items_id_seq; Type: SEQUENCE; Schema: goiardi; Owner: -
--

CREATE SEQUENCE search_items_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: search_items_id_seq; Type: SEQUENCE OWNED BY; Schema: goiardi; Owner: -
--

ALTER SEQUENCE search_items_id_seq OWNED BY search_items.id;


--
-- Name: shovey_run_streams; Type: TABLE; Schema: goiardi; Owner: -
--

CREATE TABLE shovey_run_streams (
    id bigint NOT NULL,
    shovey_run_id bigint NOT NULL,
    seq integer NOT NULL,
    output_type shovey_output,
    output text,
    is_last boolean,
    created_at timestamp with time zone NOT NULL
);


--
-- Name: shovey_run_streams_id_seq; Type: SEQUENCE; Schema: goiardi; Owner: -
--

CREATE SEQUENCE shovey_run_streams_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: shovey_run_streams_id_seq; Type: SEQUENCE OWNED BY; Schema: goiardi; Owner: -
--

ALTER SEQUENCE shovey_run_streams_id_seq OWNED BY shovey_run_streams.id;


--
-- Name: shovey_runs; Type: TABLE; Schema: goiardi; Owner: -
--

CREATE TABLE shovey_runs (
    id bigint NOT NULL,
    shovey_uuid uuid NOT NULL,
    shovey_id bigint NOT NULL,
    node_name text,
    status text,
    ack_time timestamp with time zone,
    end_time timestamp with time zone,
    error text,
    exit_status smallint
);


--
-- Name: shovey_runs_id_seq; Type: SEQUENCE; Schema: goiardi; Owner: -
--

CREATE SEQUENCE shovey_runs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: shovey_runs_id_seq; Type: SEQUENCE OWNED BY; Schema: goiardi; Owner: -
--

ALTER SEQUENCE shovey_runs_id_seq OWNED BY shovey_runs.id;


--
-- Name: shoveys; Type: TABLE; Schema: goiardi; Owner: -
--

CREATE TABLE shoveys (
    id bigint NOT NULL,
    run_id uuid NOT NULL,
    command text,
    status text,
    timeout bigint DEFAULT 300,
    quorum character varying(25) DEFAULT '100%'::character varying,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    organization_id bigint DEFAULT 1 NOT NULL
);


--
-- Name: shoveys_id_seq; Type: SEQUENCE; Schema: goiardi; Owner: -
--

CREATE SEQUENCE shoveys_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: shoveys_id_seq; Type: SEQUENCE OWNED BY; Schema: goiardi; Owner: -
--

ALTER SEQUENCE shoveys_id_seq OWNED BY shoveys.id;


--
-- Name: users; Type: TABLE; Schema: goiardi; Owner: -
--

CREATE TABLE users (
    id bigint NOT NULL,
    name text NOT NULL,
    displayname text,
    email text,
    admin boolean,
    public_key text,
    passwd character varying(128),
    salt bytea,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: users_id_seq; Type: SEQUENCE; Schema: goiardi; Owner: -
--

CREATE SEQUENCE users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: goiardi; Owner: -
--

ALTER SEQUENCE users_id_seq OWNED BY users.id;


SET search_path = sqitch, pg_catalog;

--
-- Name: changes; Type: TABLE; Schema: sqitch; Owner: -
--

CREATE TABLE changes (
    change_id text NOT NULL,
    change text NOT NULL,
    project text NOT NULL,
    note text DEFAULT ''::text NOT NULL,
    committed_at timestamp with time zone DEFAULT clock_timestamp() NOT NULL,
    committer_name text NOT NULL,
    committer_email text NOT NULL,
    planned_at timestamp with time zone NOT NULL,
    planner_name text NOT NULL,
    planner_email text NOT NULL,
    script_hash text
);


--
-- Name: TABLE changes; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON TABLE changes IS 'Tracks the changes currently deployed to the database.';


--
-- Name: COLUMN changes.change_id; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN changes.change_id IS 'Change primary key.';


--
-- Name: COLUMN changes.change; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN changes.change IS 'Name of a deployed change.';


--
-- Name: COLUMN changes.project; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN changes.project IS 'Name of the Sqitch project to which the change belongs.';


--
-- Name: COLUMN changes.note; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN changes.note IS 'Description of the change.';


--
-- Name: COLUMN changes.committed_at; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN changes.committed_at IS 'Date the change was deployed.';


--
-- Name: COLUMN changes.committer_name; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN changes.committer_name IS 'Name of the user who deployed the change.';


--
-- Name: COLUMN changes.committer_email; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN changes.committer_email IS 'Email address of the user who deployed the change.';


--
-- Name: COLUMN changes.planned_at; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN changes.planned_at IS 'Date the change was added to the plan.';


--
-- Name: COLUMN changes.planner_name; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN changes.planner_name IS 'Name of the user who planed the change.';


--
-- Name: COLUMN changes.planner_email; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN changes.planner_email IS 'Email address of the user who planned the change.';


--
-- Name: COLUMN changes.script_hash; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN changes.script_hash IS 'Deploy script SHA-1 hash.';


--
-- Name: dependencies; Type: TABLE; Schema: sqitch; Owner: -
--

CREATE TABLE dependencies (
    change_id text NOT NULL,
    type text NOT NULL,
    dependency text NOT NULL,
    dependency_id text,
    CONSTRAINT dependencies_check CHECK ((((type = 'require'::text) AND (dependency_id IS NOT NULL)) OR ((type = 'conflict'::text) AND (dependency_id IS NULL))))
);


--
-- Name: TABLE dependencies; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON TABLE dependencies IS 'Tracks the currently satisfied dependencies.';


--
-- Name: COLUMN dependencies.change_id; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN dependencies.change_id IS 'ID of the depending change.';


--
-- Name: COLUMN dependencies.type; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN dependencies.type IS 'Type of dependency.';


--
-- Name: COLUMN dependencies.dependency; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN dependencies.dependency IS 'Dependency name.';


--
-- Name: COLUMN dependencies.dependency_id; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN dependencies.dependency_id IS 'Change ID the dependency resolves to.';


--
-- Name: events; Type: TABLE; Schema: sqitch; Owner: -
--

CREATE TABLE events (
    event text NOT NULL,
    change_id text NOT NULL,
    change text NOT NULL,
    project text NOT NULL,
    note text DEFAULT ''::text NOT NULL,
    requires text[] DEFAULT '{}'::text[] NOT NULL,
    conflicts text[] DEFAULT '{}'::text[] NOT NULL,
    tags text[] DEFAULT '{}'::text[] NOT NULL,
    committed_at timestamp with time zone DEFAULT clock_timestamp() NOT NULL,
    committer_name text NOT NULL,
    committer_email text NOT NULL,
    planned_at timestamp with time zone NOT NULL,
    planner_name text NOT NULL,
    planner_email text NOT NULL,
    CONSTRAINT events_event_check CHECK ((event = ANY (ARRAY['deploy'::text, 'revert'::text, 'fail'::text, 'merge'::text])))
);


--
-- Name: TABLE events; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON TABLE events IS 'Contains full history of all deployment events.';


--
-- Name: COLUMN events.event; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN events.event IS 'Type of event.';


--
-- Name: COLUMN events.change_id; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN events.change_id IS 'Change ID.';


--
-- Name: COLUMN events.change; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN events.change IS 'Change name.';


--
-- Name: COLUMN events.project; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN events.project IS 'Name of the Sqitch project to which the change belongs.';


--
-- Name: COLUMN events.note; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN events.note IS 'Description of the change.';


--
-- Name: COLUMN events.requires; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN events.requires IS 'Array of the names of required changes.';


--
-- Name: COLUMN events.conflicts; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN events.conflicts IS 'Array of the names of conflicting changes.';


--
-- Name: COLUMN events.tags; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN events.tags IS 'Tags associated with the change.';


--
-- Name: COLUMN events.committed_at; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN events.committed_at IS 'Date the event was committed.';


--
-- Name: COLUMN events.committer_name; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN events.committer_name IS 'Name of the user who committed the event.';


--
-- Name: COLUMN events.committer_email; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN events.committer_email IS 'Email address of the user who committed the event.';


--
-- Name: COLUMN events.planned_at; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN events.planned_at IS 'Date the event was added to the plan.';


--
-- Name: COLUMN events.planner_name; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN events.planner_name IS 'Name of the user who planed the change.';


--
-- Name: COLUMN events.planner_email; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN events.planner_email IS 'Email address of the user who plan planned the change.';


--
-- Name: projects; Type: TABLE; Schema: sqitch; Owner: -
--

CREATE TABLE projects (
    project text NOT NULL,
    uri text,
    created_at timestamp with time zone DEFAULT clock_timestamp() NOT NULL,
    creator_name text NOT NULL,
    creator_email text NOT NULL
);


--
-- Name: TABLE projects; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON TABLE projects IS 'Sqitch projects deployed to this database.';


--
-- Name: COLUMN projects.project; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN projects.project IS 'Unique Name of a project.';


--
-- Name: COLUMN projects.uri; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN projects.uri IS 'Optional project URI';


--
-- Name: COLUMN projects.created_at; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN projects.created_at IS 'Date the project was added to the database.';


--
-- Name: COLUMN projects.creator_name; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN projects.creator_name IS 'Name of the user who added the project.';


--
-- Name: COLUMN projects.creator_email; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN projects.creator_email IS 'Email address of the user who added the project.';


--
-- Name: releases; Type: TABLE; Schema: sqitch; Owner: -
--

CREATE TABLE releases (
    version real NOT NULL,
    installed_at timestamp with time zone DEFAULT clock_timestamp() NOT NULL,
    installer_name text NOT NULL,
    installer_email text NOT NULL
);


--
-- Name: TABLE releases; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON TABLE releases IS 'Sqitch registry releases.';


--
-- Name: COLUMN releases.version; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN releases.version IS 'Version of the Sqitch registry.';


--
-- Name: COLUMN releases.installed_at; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN releases.installed_at IS 'Date the registry release was installed.';


--
-- Name: COLUMN releases.installer_name; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN releases.installer_name IS 'Name of the user who installed the registry release.';


--
-- Name: COLUMN releases.installer_email; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN releases.installer_email IS 'Email address of the user who installed the registry release.';


--
-- Name: tags; Type: TABLE; Schema: sqitch; Owner: -
--

CREATE TABLE tags (
    tag_id text NOT NULL,
    tag text NOT NULL,
    project text NOT NULL,
    change_id text NOT NULL,
    note text DEFAULT ''::text NOT NULL,
    committed_at timestamp with time zone DEFAULT clock_timestamp() NOT NULL,
    committer_name text NOT NULL,
    committer_email text NOT NULL,
    planned_at timestamp with time zone NOT NULL,
    planner_name text NOT NULL,
    planner_email text NOT NULL
);


--
-- Name: TABLE tags; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON TABLE tags IS 'Tracks the tags currently applied to the database.';


--
-- Name: COLUMN tags.tag_id; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN tags.tag_id IS 'Tag primary key.';


--
-- Name: COLUMN tags.tag; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN tags.tag IS 'Project-unique tag name.';


--
-- Name: COLUMN tags.project; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN tags.project IS 'Name of the Sqitch project to which the tag belongs.';


--
-- Name: COLUMN tags.change_id; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN tags.change_id IS 'ID of last change deployed before the tag was applied.';


--
-- Name: COLUMN tags.note; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN tags.note IS 'Description of the tag.';


--
-- Name: COLUMN tags.committed_at; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN tags.committed_at IS 'Date the tag was applied to the database.';


--
-- Name: COLUMN tags.committer_name; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN tags.committer_name IS 'Name of the user who applied the tag.';


--
-- Name: COLUMN tags.committer_email; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN tags.committer_email IS 'Email address of the user who applied the tag.';


--
-- Name: COLUMN tags.planned_at; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN tags.planned_at IS 'Date the tag was added to the plan.';


--
-- Name: COLUMN tags.planner_name; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN tags.planner_name IS 'Name of the user who planed the tag.';


--
-- Name: COLUMN tags.planner_email; Type: COMMENT; Schema: sqitch; Owner: -
--

COMMENT ON COLUMN tags.planner_email IS 'Email address of the user who planned the tag.';


SET search_path = goiardi, pg_catalog;

--
-- Name: id; Type: DEFAULT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY clients ALTER COLUMN id SET DEFAULT nextval('clients_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY cookbook_versions ALTER COLUMN id SET DEFAULT nextval('cookbook_versions_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY cookbooks ALTER COLUMN id SET DEFAULT nextval('cookbooks_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY data_bag_items ALTER COLUMN id SET DEFAULT nextval('data_bag_items_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY data_bags ALTER COLUMN id SET DEFAULT nextval('data_bags_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY environments ALTER COLUMN id SET DEFAULT nextval('environments_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY file_checksums ALTER COLUMN id SET DEFAULT nextval('file_checksums_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY log_infos ALTER COLUMN id SET DEFAULT nextval('log_infos_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY node_statuses ALTER COLUMN id SET DEFAULT nextval('node_statuses_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY nodes ALTER COLUMN id SET DEFAULT nextval('nodes_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY organizations ALTER COLUMN id SET DEFAULT nextval('organizations_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY reports ALTER COLUMN id SET DEFAULT nextval('reports_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY roles ALTER COLUMN id SET DEFAULT nextval('roles_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY sandboxes ALTER COLUMN id SET DEFAULT nextval('sandboxes_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY search_collections ALTER COLUMN id SET DEFAULT nextval('search_collections_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY search_items ALTER COLUMN id SET DEFAULT nextval('search_items_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY shovey_run_streams ALTER COLUMN id SET DEFAULT nextval('shovey_run_streams_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY shovey_runs ALTER COLUMN id SET DEFAULT nextval('shovey_runs_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY shoveys ALTER COLUMN id SET DEFAULT nextval('shoveys_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY users ALTER COLUMN id SET DEFAULT nextval('users_id_seq'::regclass);


--
-- Data for Name: clients; Type: TABLE DATA; Schema: goiardi; Owner: -
--

COPY clients (id, name, nodename, validator, admin, organization_id, public_key, certificate, created_at, updated_at) FROM stdin;
\.


--
-- Name: clients_id_seq; Type: SEQUENCE SET; Schema: goiardi; Owner: -
--

SELECT pg_catalog.setval('clients_id_seq', 1, false);


--
-- Data for Name: cookbook_versions; Type: TABLE DATA; Schema: goiardi; Owner: -
--

COPY cookbook_versions (id, cookbook_id, major_ver, minor_ver, patch_ver, frozen, metadata, definitions, libraries, attributes, recipes, providers, resources, templates, root_files, files, created_at, updated_at) FROM stdin;
\.


--
-- Name: cookbook_versions_id_seq; Type: SEQUENCE SET; Schema: goiardi; Owner: -
--

SELECT pg_catalog.setval('cookbook_versions_id_seq', 1, false);


--
-- Data for Name: cookbooks; Type: TABLE DATA; Schema: goiardi; Owner: -
--

COPY cookbooks (id, name, organization_id, created_at, updated_at) FROM stdin;
\.


--
-- Name: cookbooks_id_seq; Type: SEQUENCE SET; Schema: goiardi; Owner: -
--

SELECT pg_catalog.setval('cookbooks_id_seq', 1, false);


--
-- Data for Name: data_bag_items; Type: TABLE DATA; Schema: goiardi; Owner: -
--

COPY data_bag_items (id, name, orig_name, data_bag_id, raw_data, created_at, updated_at) FROM stdin;
\.


--
-- Name: data_bag_items_id_seq; Type: SEQUENCE SET; Schema: goiardi; Owner: -
--

SELECT pg_catalog.setval('data_bag_items_id_seq', 1, false);


--
-- Data for Name: data_bags; Type: TABLE DATA; Schema: goiardi; Owner: -
--

COPY data_bags (id, name, organization_id, created_at, updated_at) FROM stdin;
\.


--
-- Name: data_bags_id_seq; Type: SEQUENCE SET; Schema: goiardi; Owner: -
--

SELECT pg_catalog.setval('data_bags_id_seq', 1, false);


--
-- Data for Name: environments; Type: TABLE DATA; Schema: goiardi; Owner: -
--

COPY environments (id, name, organization_id, description, default_attr, override_attr, cookbook_vers, created_at, updated_at) FROM stdin;
1	_default	1	The default Chef environment	\N	\N	\N	2016-10-24 01:36:00.184006-07	2016-10-24 01:36:00.184006-07
\.


--
-- Name: environments_id_seq; Type: SEQUENCE SET; Schema: goiardi; Owner: -
--

SELECT pg_catalog.setval('environments_id_seq', 1, false);


--
-- Data for Name: file_checksums; Type: TABLE DATA; Schema: goiardi; Owner: -
--

COPY file_checksums (id, organization_id, checksum) FROM stdin;
\.


--
-- Name: file_checksums_id_seq; Type: SEQUENCE SET; Schema: goiardi; Owner: -
--

SELECT pg_catalog.setval('file_checksums_id_seq', 1, false);


--
-- Data for Name: log_infos; Type: TABLE DATA; Schema: goiardi; Owner: -
--

COPY log_infos (id, actor_id, actor_info, actor_type, organization_id, "time", action, object_type, object_name, extended_info) FROM stdin;
\.


--
-- Name: log_infos_id_seq; Type: SEQUENCE SET; Schema: goiardi; Owner: -
--

SELECT pg_catalog.setval('log_infos_id_seq', 1, false);


--
-- Data for Name: node_statuses; Type: TABLE DATA; Schema: goiardi; Owner: -
--

COPY node_statuses (id, node_id, status, updated_at) FROM stdin;
\.


--
-- Name: node_statuses_id_seq; Type: SEQUENCE SET; Schema: goiardi; Owner: -
--

SELECT pg_catalog.setval('node_statuses_id_seq', 1, false);


--
-- Data for Name: nodes; Type: TABLE DATA; Schema: goiardi; Owner: -
--

COPY nodes (id, name, organization_id, chef_environment, run_list, automatic_attr, normal_attr, default_attr, override_attr, created_at, updated_at, is_down) FROM stdin;
\.


--
-- Name: nodes_id_seq; Type: SEQUENCE SET; Schema: goiardi; Owner: -
--

SELECT pg_catalog.setval('nodes_id_seq', 1, false);


--
-- Data for Name: organizations; Type: TABLE DATA; Schema: goiardi; Owner: -
--

COPY organizations (id, name, description, created_at, updated_at) FROM stdin;
1	default	\N	2016-10-24 01:36:00.437892-07	2016-10-24 01:36:00.437892-07
\.


--
-- Name: organizations_id_seq; Type: SEQUENCE SET; Schema: goiardi; Owner: -
--

SELECT pg_catalog.setval('organizations_id_seq', 1, true);


--
-- Data for Name: reports; Type: TABLE DATA; Schema: goiardi; Owner: -
--

COPY reports (id, run_id, node_name, organization_id, start_time, end_time, total_res_count, status, run_list, resources, data, created_at, updated_at) FROM stdin;
\.


--
-- Name: reports_id_seq; Type: SEQUENCE SET; Schema: goiardi; Owner: -
--

SELECT pg_catalog.setval('reports_id_seq', 1, false);


--
-- Data for Name: roles; Type: TABLE DATA; Schema: goiardi; Owner: -
--

COPY roles (id, name, organization_id, description, run_list, env_run_lists, default_attr, override_attr, created_at, updated_at) FROM stdin;
\.


--
-- Name: roles_id_seq; Type: SEQUENCE SET; Schema: goiardi; Owner: -
--

SELECT pg_catalog.setval('roles_id_seq', 1, false);


--
-- Data for Name: sandboxes; Type: TABLE DATA; Schema: goiardi; Owner: -
--

COPY sandboxes (id, sbox_id, organization_id, creation_time, checksums, completed) FROM stdin;
\.


--
-- Name: sandboxes_id_seq; Type: SEQUENCE SET; Schema: goiardi; Owner: -
--

SELECT pg_catalog.setval('sandboxes_id_seq', 1, false);


--
-- Data for Name: search_collections; Type: TABLE DATA; Schema: goiardi; Owner: -
--

COPY search_collections (id, organization_id, name) FROM stdin;
\.


--
-- Name: search_collections_id_seq; Type: SEQUENCE SET; Schema: goiardi; Owner: -
--

SELECT pg_catalog.setval('search_collections_id_seq', 1, false);


--
-- Data for Name: search_items; Type: TABLE DATA; Schema: goiardi; Owner: -
--

COPY search_items (id, organization_id, search_collection_id, item_name, value, path) FROM stdin;
\.


--
-- Name: search_items_id_seq; Type: SEQUENCE SET; Schema: goiardi; Owner: -
--

SELECT pg_catalog.setval('search_items_id_seq', 1, false);


--
-- Data for Name: shovey_run_streams; Type: TABLE DATA; Schema: goiardi; Owner: -
--

COPY shovey_run_streams (id, shovey_run_id, seq, output_type, output, is_last, created_at) FROM stdin;
\.


--
-- Name: shovey_run_streams_id_seq; Type: SEQUENCE SET; Schema: goiardi; Owner: -
--

SELECT pg_catalog.setval('shovey_run_streams_id_seq', 1, false);


--
-- Data for Name: shovey_runs; Type: TABLE DATA; Schema: goiardi; Owner: -
--

COPY shovey_runs (id, shovey_uuid, shovey_id, node_name, status, ack_time, end_time, error, exit_status) FROM stdin;
\.


--
-- Name: shovey_runs_id_seq; Type: SEQUENCE SET; Schema: goiardi; Owner: -
--

SELECT pg_catalog.setval('shovey_runs_id_seq', 1, false);


--
-- Data for Name: shoveys; Type: TABLE DATA; Schema: goiardi; Owner: -
--

COPY shoveys (id, run_id, command, status, timeout, quorum, created_at, updated_at, organization_id) FROM stdin;
\.


--
-- Name: shoveys_id_seq; Type: SEQUENCE SET; Schema: goiardi; Owner: -
--

SELECT pg_catalog.setval('shoveys_id_seq', 1, false);


--
-- Data for Name: users; Type: TABLE DATA; Schema: goiardi; Owner: -
--

COPY users (id, name, displayname, email, admin, public_key, passwd, salt, created_at, updated_at) FROM stdin;
\.


--
-- Name: users_id_seq; Type: SEQUENCE SET; Schema: goiardi; Owner: -
--

SELECT pg_catalog.setval('users_id_seq', 1, false);


SET search_path = sqitch, pg_catalog;

--
-- Data for Name: changes; Type: TABLE DATA; Schema: sqitch; Owner: -
--

COPY changes (change_id, change, project, note, committed_at, committer_name, committer_email, planned_at, planner_name, planner_email, script_hash) FROM stdin;
c89b0e25c808b327036c88e6c9750c7526314c86	goiardi_schema	goiardi_postgres	Add schema for goiardi-postgres	2016-10-24 01:36:00.169744-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-27 14:09:07-07	Jeremy Bingham	jbingham@gmail.com	6ec25c903515e34857bb9090bd1a87e9e927e911
367c28670efddf25455b9fd33c23a5a278b08bb4	environments	goiardi_postgres	Environments for postgres	2016-10-24 01:36:00.192931-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 00:40:11-07	Jeremy Bingham	jbingham@gmail.com	14071321aa065ea2fc16394ac7eccac4e17f871e
911c456769628c817340ee77fc8d2b7c1d697782	nodes	goiardi_postgres	Create node table	2016-10-24 01:36:00.217871-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 10:37:46-07	Jeremy Bingham	jbingham@gmail.com	32109bb49aa36a94c712b7acda79c47ff8ddec25
faa3571aa479de60f25785e707433b304ba3d2c7	clients	goiardi_postgres	Create client table	2016-10-24 01:36:00.240711-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 11:05:33-07	Jeremy Bingham	jbingham@gmail.com	1e6f62a97fbc07211d2968372284d63dd4aa5991
bb82d8869ffca8ba3d03a1502c50dbb3eee7a2e0	users	goiardi_postgres	Create user table	2016-10-24 01:36:00.263578-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 11:15:02-07	Jeremy Bingham	jbingham@gmail.com	fa249b238927ea80b6812de8d14c3d392b16ac95
138bc49d92c0bbb024cea41532a656f2d7f9b072	cookbooks	goiardi_postgres	Create cookbook  table	2016-10-24 01:36:00.285625-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 11:27:27-07	Jeremy Bingham	jbingham@gmail.com	566949982dcc9c9795e7764c67b0437cc0f8422b
f529038064a0259bdecbdab1f9f665e17ddb6136	cookbook_versions	goiardi_postgres	Create cookbook versions table	2016-10-24 01:36:00.308155-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 11:31:34-07	Jeremy Bingham	jbingham@gmail.com	9566c4794ddb4258426ce03bbeb03529fb57c240
85483913f96710c1267c6abacb6568cef9327f15	data_bags	goiardi_postgres	Create cookbook data bags table	2016-10-24 01:36:00.329717-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 11:42:04-07	Jeremy Bingham	jbingham@gmail.com	17f8dbb44a32f1c0a01f9b3dde21dd2a40ecb53b
feddf91b62caed36c790988bd29222591980433b	data_bag_items	goiardi_postgres	Create data bag items table	2016-10-24 01:36:00.352727-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 12:02:31-07	Jeremy Bingham	jbingham@gmail.com	b928bdba9937cdfbc3bcb20e2808674485d6d22e
6a4489d9436ba1541d272700b303410cc906b08f	roles	goiardi_postgres	Create roles table	2016-10-24 01:36:00.375924-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 12:09:28-07	Jeremy Bingham	jbingham@gmail.com	67bbe92a7725f64280d6b031f9726997045d032e
c4b32778f2911930f583ce15267aade320ac4dcd	sandboxes	goiardi_postgres	Create sandboxes table	2016-10-24 01:36:00.39715-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 12:14:48-07	Jeremy Bingham	jbingham@gmail.com	06db5b16a86236a68401cdb4f6fe8f83492ec294
81003655b93b41359804027fc202788aa0ddd9a9	log_infos	goiardi_postgres	Create log_infos table	2016-10-24 01:36:00.42422-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 12:19:10-07	Jeremy Bingham	jbingham@gmail.com	c3e322a1c9a2a6fdb37d116e47c02c5ad91b245b
fce5b7aeed2ad742de1309d7841577cff19475a7	organizations	goiardi_postgres	Create organizations table	2016-10-24 01:36:00.44727-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 12:46:28-07	Jeremy Bingham	jbingham@gmail.com	0702ea36c9d8db6a336d8a54963968e8bbb797bd
f2621482d1c130ea8fee15d09f966685409bf67c	file_checksums	goiardi_postgres	Create file checksums table	2016-10-24 01:36:00.468422-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 12:49:19-07	Jeremy Bingham	jbingham@gmail.com	4e2aef9e549ac3fdaf5394bc53a690b8f9eb426a
db1eb360cd5e6449a468ceb781d82b45dafb5c2d	reports	goiardi_postgres	Create reports table	2016-10-24 01:36:00.493465-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 13:02:49-07	Jeremy Bingham	jbingham@gmail.com	70ed195e5aa230c82883980ceb9d193c9d288039
c8b38382f7e5a18f36c621327f59205aa8aa9849	client_insert_duplicate	goiardi_postgres	Function to emulate insert ... on duplicate update for clients	2016-10-24 01:36:00.512661-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 23:00:04-07	Jeremy Bingham	jbingham@gmail.com	81a3b18bd917955e8941d013e542bd31a64d707b
30774a960a0efb6adfbb1d526b8cdb1a45c7d039	client_rename	goiardi_postgres	Function to rename clients	2016-10-24 01:36:00.530852-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-30 10:22:50-07	Jeremy Bingham	jbingham@gmail.com	733c2530ed7f7a84a5a7eded876a4b5f5a7fd5d7
2d1fdc8128b0632e798df7346e76f122ed5915ec	user_insert_duplicate	goiardi_postgres	Function to emulate insert ... on duplicate update for clients	2016-10-24 01:36:00.548853-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-30 13:07:46-07	Jeremy Bingham	jbingham@gmail.com	157af7fecc8228af6cd0638188510069cb2785ca
f336c149ab32530c9c6ae4408c11558a635f39a1	user_rename	goiardi_postgres	Function to rename users	2016-10-24 01:36:00.566655-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-30 13:15:45-07	Jeremy Bingham	jbingham@gmail.com	8721b0d316f9951a197656ae7a595db952a5b753
841a7d554d44f9d0d0b8a1a5a9d0a06ce71a2453	cookbook_insert_update	goiardi_postgres	Cookbook insert/update	2016-10-24 01:36:00.584748-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-30 23:55:23-07	Jeremy Bingham	jbingham@gmail.com	0157bb4d66921441eba381b7e4706407791356ad
085e2f6281914c9fa6521d59fea81f16c106b59f	cookbook_versions_insert_update	goiardi_postgres	Cookbook versions insert/update	2016-10-24 01:36:00.602748-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-30 23:56:05-07	Jeremy Bingham	jbingham@gmail.com	02acdeff93d07edf150ef2cab00372009e0119ee
04bea39d649e4187d9579bd946fd60f760240d10	data_bag_insert_update	goiardi_postgres	Insert/update data bags	2016-10-24 01:36:00.620974-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-31 23:25:44-07	Jeremy Bingham	jbingham@gmail.com	d2c19ce6757f915987e12fa4fd17f71be5409a80
092885e8b5d94a9c1834bf309e02dc0f955ff053	environment_insert_update	goiardi_postgres	Insert/update environments	2016-10-24 01:36:00.639875-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-01 12:55:34-07	Jeremy Bingham	jbingham@gmail.com	e0807c6d3471284d1712fb156c98c05a1770460b
6d9587fa4275827c93ca9d7e0166ad1887b76cad	file_checksum_insert_ignore	goiardi_postgres	Insert ignore for file checksums	2016-10-24 01:36:00.657793-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-01 23:13:48-07	Jeremy Bingham	jbingham@gmail.com	9a3f07ed5472dbd1a00d861f2f5b4413877e5fb3
82a95e5e6cbd8ba51fea33506e1edb2a12e37a92	node_insert_update	goiardi_postgres	Insert/update for nodes	2016-10-24 01:36:00.676823-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-01 23:25:20-07	Jeremy Bingham	jbingham@gmail.com	d2eb971a3888d7e5291875e57ac170b9a98e34a9
d052a8267a6512581e5cab1f89a2456f279727b9	report_insert_update	goiardi_postgres	Insert/update for reports	2016-10-24 01:36:00.694263-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-02 10:10:25-07	Jeremy Bingham	jbingham@gmail.com	808c27804d8f42bd21935a8da0e48480f187946f
acf76029633d50febbec7c4763b7173078eddaf7	role_insert_update	goiardi_postgres	Insert/update for roles	2016-10-24 01:36:00.712957-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-02 10:27:32-07	Jeremy Bingham	jbingham@gmail.com	7d90b6826a43b5751481f6f1ad6411f80686e15e
b8ef36df686397ecb0fe67eb097e84aa0d78ac6b	sandbox_insert_update	goiardi_postgres	Insert/update for sandboxes	2016-10-24 01:36:00.731156-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-02 10:34:39-07	Jeremy Bingham	jbingham@gmail.com	12656c98699f59034eb4b692e8fe517705c072df
93dbbda50a25da0a586e89ccee8fcfa2ddcb7c64	data_bag_item_insert	goiardi_postgres	Insert for data bag items	2016-10-24 01:36:00.749432-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-02 14:03:22-07	Jeremy Bingham	jbingham@gmail.com	8d32ce4373fbb0141ec8aeba5dc0e6b178a507af
c80c561c22f6e139165cdb338c7ce6fff8ff268d	bytea_to_json	goiardi_postgres	Change most postgres bytea fields to json, because in this peculiar case json is way faster than gob	2016-10-24 01:36:00.807371-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-20 02:41:22-07	Jeremy Bingham	jbingham@gmail.com	f8ae9cbad0b78031ec802537e37487c47e38d7d5
9966894e0fc0da573243f6a3c0fc1432a2b63043	joined_cookbkook_version	goiardi_postgres	a convenient view for joined versions for cookbook versions, adapted from erchef's joined_cookbook_version	2016-10-24 01:36:00.825939-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-20 03:21:28-07	Jeremy Bingham	jbingham@gmail.com	1a1fb217309986790f82be8e0266e601b8ec18dc
163ba4a496b9b4210d335e0e4ea5368a9ea8626c	node_statuses	goiardi_postgres	Create node_status table for node statuses	2016-10-24 01:36:00.8491-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-10 23:01:54-07	Jeremy Bingham	jeremy@terqa.local	a61ba9d203c458ccd2a1150caae1239b85ef0334
8bb822f391b499585cfb2fc7248be469b0200682	node_status_insert	goiardi_postgres	insert function for node_statuses	2016-10-24 01:36:00.867577-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-11 00:01:31-07	Jeremy Bingham	jeremy@terqa.local	95a2c1c5f017868c10062006284da4c0aa9a69d7
7c429aac08527adc774767584201f668408b04a6	add_down_column_nodes	goiardi_postgres	Add is_down column to the nodes table	2016-10-24 01:36:00.891045-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-15 20:18:05-07	Jeremy Bingham	jbingham@gmail.com	f0a4e7d49cd2c396326b4f3ddef46c968e8174cd
82bcace325dbdc905eb6e677f800d14a0506a216	shovey	goiardi_postgres	add shovey tables	2016-10-24 01:36:00.928682-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-15 22:07:12-07	Jeremy Bingham	jeremy@terqa.local	ccde488018934b941a95d96a150265520cfa4d25
62046d2fb96bbaedce2406252d312766452551c0	node_latest_statuses	goiardi_postgres	Add a view to easily get nodes by their latest status	2016-10-24 01:36:00.947295-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-26 13:32:02-07	Jeremy Bingham	jbingham@gmail.com	f5c6a6785ded2b5bddfac79b3e88ecd1ac323759
68f90e1fd2aac6a117d7697626741a02b8d0ebbe	shovey_insert_update	goiardi_postgres	insert/update functions for shovey	2016-10-24 01:36:00.966335-07	Jeremy Bingham	jeremy@goiardi.gl	2014-08-27 00:46:20-07	Jeremy Bingham	jbingham@gmail.com	74df17dfb21d951347df4543c963d300a2f514ca
6f7aa2430e01cf33715828f1957d072cd5006d1c	ltree	goiardi_postgres	Add tables for ltree search for postgres	2016-10-24 01:36:01.019465-07	Jeremy Bingham	jeremy@goiardi.gl	2015-04-10 23:21:26-07	Jeremy Bingham	jeremy@goiardi.gl	124d1a4e9b8ceb1a87defd605da75c97c4a919e3
e7eb33b00d2fb6302e0c3979e9cac6fb80da377e	ltree_del_col	goiardi_postgres	procedure for deleting search collections	2016-10-24 01:36:01.038487-07	Jeremy Bingham	jeremy@goiardi.gl	2015-04-12 12:33:15-07	Jeremy Bingham	jeremy@goiardi.gl	bdb43f64834424709550172bb395f4539a98de4d
f49decbb15053ec5691093568450f642578ca460	ltree_del_item	goiardi_postgres	procedure for deleting search items	2016-10-24 01:36:01.055696-07	Jeremy Bingham	jeremy@goiardi.gl	2015-04-12 13:03:50-07	Jeremy Bingham	jeremy@goiardi.gl	443bbe855ebc934501a28922898771c1590d9bd9
d87c4dc108d4fa90942cc3bab8e619a58aef3d2d	jsonb	goiardi_postgres	Switch from json to jsonb columns. Will require using postgres 9.4+.	2016-10-24 01:36:01.113393-07	Jeremy Bingham	jeremy@goiardi.gl	2016-09-09 01:17:31-07	Jeremy Bingham	jeremy@eridu.local	fe7c2d072328101c3f343613e869d753842fcfe2
\.


--
-- Data for Name: dependencies; Type: TABLE DATA; Schema: sqitch; Owner: -
--

COPY dependencies (change_id, type, dependency, dependency_id) FROM stdin;
367c28670efddf25455b9fd33c23a5a278b08bb4	require	goiardi_schema	c89b0e25c808b327036c88e6c9750c7526314c86
911c456769628c817340ee77fc8d2b7c1d697782	require	goiardi_schema	c89b0e25c808b327036c88e6c9750c7526314c86
faa3571aa479de60f25785e707433b304ba3d2c7	require	goiardi_schema	c89b0e25c808b327036c88e6c9750c7526314c86
bb82d8869ffca8ba3d03a1502c50dbb3eee7a2e0	require	goiardi_schema	c89b0e25c808b327036c88e6c9750c7526314c86
138bc49d92c0bbb024cea41532a656f2d7f9b072	require	goiardi_schema	c89b0e25c808b327036c88e6c9750c7526314c86
f529038064a0259bdecbdab1f9f665e17ddb6136	require	cookbooks	138bc49d92c0bbb024cea41532a656f2d7f9b072
f529038064a0259bdecbdab1f9f665e17ddb6136	require	goiardi_schema	c89b0e25c808b327036c88e6c9750c7526314c86
85483913f96710c1267c6abacb6568cef9327f15	require	goiardi_schema	c89b0e25c808b327036c88e6c9750c7526314c86
feddf91b62caed36c790988bd29222591980433b	require	data_bags	85483913f96710c1267c6abacb6568cef9327f15
feddf91b62caed36c790988bd29222591980433b	require	goiardi_schema	c89b0e25c808b327036c88e6c9750c7526314c86
6a4489d9436ba1541d272700b303410cc906b08f	require	goiardi_schema	c89b0e25c808b327036c88e6c9750c7526314c86
c4b32778f2911930f583ce15267aade320ac4dcd	require	goiardi_schema	c89b0e25c808b327036c88e6c9750c7526314c86
81003655b93b41359804027fc202788aa0ddd9a9	require	clients	faa3571aa479de60f25785e707433b304ba3d2c7
81003655b93b41359804027fc202788aa0ddd9a9	require	users	bb82d8869ffca8ba3d03a1502c50dbb3eee7a2e0
81003655b93b41359804027fc202788aa0ddd9a9	require	goiardi_schema	c89b0e25c808b327036c88e6c9750c7526314c86
c8b38382f7e5a18f36c621327f59205aa8aa9849	require	clients	faa3571aa479de60f25785e707433b304ba3d2c7
c8b38382f7e5a18f36c621327f59205aa8aa9849	require	goiardi_schema	c89b0e25c808b327036c88e6c9750c7526314c86
30774a960a0efb6adfbb1d526b8cdb1a45c7d039	require	clients	faa3571aa479de60f25785e707433b304ba3d2c7
30774a960a0efb6adfbb1d526b8cdb1a45c7d039	require	goiardi_schema	c89b0e25c808b327036c88e6c9750c7526314c86
2d1fdc8128b0632e798df7346e76f122ed5915ec	require	users	bb82d8869ffca8ba3d03a1502c50dbb3eee7a2e0
2d1fdc8128b0632e798df7346e76f122ed5915ec	require	goiardi_schema	c89b0e25c808b327036c88e6c9750c7526314c86
f336c149ab32530c9c6ae4408c11558a635f39a1	require	users	bb82d8869ffca8ba3d03a1502c50dbb3eee7a2e0
f336c149ab32530c9c6ae4408c11558a635f39a1	require	goiardi_schema	c89b0e25c808b327036c88e6c9750c7526314c86
841a7d554d44f9d0d0b8a1a5a9d0a06ce71a2453	require	cookbooks	138bc49d92c0bbb024cea41532a656f2d7f9b072
841a7d554d44f9d0d0b8a1a5a9d0a06ce71a2453	require	goiardi_schema	c89b0e25c808b327036c88e6c9750c7526314c86
085e2f6281914c9fa6521d59fea81f16c106b59f	require	cookbook_versions	f529038064a0259bdecbdab1f9f665e17ddb6136
085e2f6281914c9fa6521d59fea81f16c106b59f	require	goiardi_schema	c89b0e25c808b327036c88e6c9750c7526314c86
04bea39d649e4187d9579bd946fd60f760240d10	require	data_bags	85483913f96710c1267c6abacb6568cef9327f15
04bea39d649e4187d9579bd946fd60f760240d10	require	goiardi_schema	c89b0e25c808b327036c88e6c9750c7526314c86
092885e8b5d94a9c1834bf309e02dc0f955ff053	require	environments	367c28670efddf25455b9fd33c23a5a278b08bb4
092885e8b5d94a9c1834bf309e02dc0f955ff053	require	goiardi_schema	c89b0e25c808b327036c88e6c9750c7526314c86
6d9587fa4275827c93ca9d7e0166ad1887b76cad	require	file_checksums	f2621482d1c130ea8fee15d09f966685409bf67c
6d9587fa4275827c93ca9d7e0166ad1887b76cad	require	goiardi_schema	c89b0e25c808b327036c88e6c9750c7526314c86
82a95e5e6cbd8ba51fea33506e1edb2a12e37a92	require	nodes	911c456769628c817340ee77fc8d2b7c1d697782
82a95e5e6cbd8ba51fea33506e1edb2a12e37a92	require	goiardi_schema	c89b0e25c808b327036c88e6c9750c7526314c86
d052a8267a6512581e5cab1f89a2456f279727b9	require	reports	db1eb360cd5e6449a468ceb781d82b45dafb5c2d
d052a8267a6512581e5cab1f89a2456f279727b9	require	goiardi_schema	c89b0e25c808b327036c88e6c9750c7526314c86
acf76029633d50febbec7c4763b7173078eddaf7	require	roles	6a4489d9436ba1541d272700b303410cc906b08f
acf76029633d50febbec7c4763b7173078eddaf7	require	goiardi_schema	c89b0e25c808b327036c88e6c9750c7526314c86
b8ef36df686397ecb0fe67eb097e84aa0d78ac6b	require	sandboxes	c4b32778f2911930f583ce15267aade320ac4dcd
b8ef36df686397ecb0fe67eb097e84aa0d78ac6b	require	goiardi_schema	c89b0e25c808b327036c88e6c9750c7526314c86
93dbbda50a25da0a586e89ccee8fcfa2ddcb7c64	require	data_bag_items	feddf91b62caed36c790988bd29222591980433b
93dbbda50a25da0a586e89ccee8fcfa2ddcb7c64	require	data_bags	85483913f96710c1267c6abacb6568cef9327f15
93dbbda50a25da0a586e89ccee8fcfa2ddcb7c64	require	goiardi_schema	c89b0e25c808b327036c88e6c9750c7526314c86
163ba4a496b9b4210d335e0e4ea5368a9ea8626c	require	nodes	911c456769628c817340ee77fc8d2b7c1d697782
8bb822f391b499585cfb2fc7248be469b0200682	require	node_statuses	163ba4a496b9b4210d335e0e4ea5368a9ea8626c
7c429aac08527adc774767584201f668408b04a6	require	nodes	911c456769628c817340ee77fc8d2b7c1d697782
62046d2fb96bbaedce2406252d312766452551c0	require	node_statuses	163ba4a496b9b4210d335e0e4ea5368a9ea8626c
68f90e1fd2aac6a117d7697626741a02b8d0ebbe	require	shovey	82bcace325dbdc905eb6e677f800d14a0506a216
\.


--
-- Data for Name: events; Type: TABLE DATA; Schema: sqitch; Owner: -
--

COPY events (event, change_id, change, project, note, requires, conflicts, tags, committed_at, committer_name, committer_email, planned_at, planner_name, planner_email) FROM stdin;
deploy	c89b0e25c808b327036c88e6c9750c7526314c86	goiardi_schema	goiardi_postgres	Add schema for goiardi-postgres	{}	{}	{}	2014-09-24 21:30:12.905964-07	Jeremy Bingham	jbingham@gmail.com	2014-05-27 14:09:07-07	Jeremy Bingham	jbingham@gmail.com
deploy	367c28670efddf25455b9fd33c23a5a278b08bb4	environments	goiardi_postgres	Environments for postgres	{goiardi_schema}	{}	{}	2014-09-24 21:30:12.928508-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 00:40:11-07	Jeremy Bingham	jbingham@gmail.com
deploy	911c456769628c817340ee77fc8d2b7c1d697782	nodes	goiardi_postgres	Create node table	{goiardi_schema}	{}	{}	2014-09-24 21:30:12.94921-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 10:37:46-07	Jeremy Bingham	jbingham@gmail.com
deploy	faa3571aa479de60f25785e707433b304ba3d2c7	clients	goiardi_postgres	Create client table	{goiardi_schema}	{}	{}	2014-09-24 21:30:12.969599-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 11:05:33-07	Jeremy Bingham	jbingham@gmail.com
deploy	bb82d8869ffca8ba3d03a1502c50dbb3eee7a2e0	users	goiardi_postgres	Create user table	{goiardi_schema}	{}	{}	2014-09-24 21:30:12.995205-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 11:15:02-07	Jeremy Bingham	jbingham@gmail.com
deploy	138bc49d92c0bbb024cea41532a656f2d7f9b072	cookbooks	goiardi_postgres	Create cookbook  table	{goiardi_schema}	{}	{}	2014-09-24 21:30:13.013206-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 11:27:27-07	Jeremy Bingham	jbingham@gmail.com
deploy	f529038064a0259bdecbdab1f9f665e17ddb6136	cookbook_versions	goiardi_postgres	Create cookbook versions table	{cookbooks,goiardi_schema}	{}	{}	2014-09-24 21:30:13.031793-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 11:31:34-07	Jeremy Bingham	jbingham@gmail.com
deploy	85483913f96710c1267c6abacb6568cef9327f15	data_bags	goiardi_postgres	Create cookbook data bags table	{goiardi_schema}	{}	{}	2014-09-24 21:30:13.050839-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 11:42:04-07	Jeremy Bingham	jbingham@gmail.com
deploy	feddf91b62caed36c790988bd29222591980433b	data_bag_items	goiardi_postgres	Create data bag items table	{data_bags,goiardi_schema}	{}	{}	2014-09-24 21:30:13.070789-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 12:02:31-07	Jeremy Bingham	jbingham@gmail.com
deploy	6a4489d9436ba1541d272700b303410cc906b08f	roles	goiardi_postgres	Create roles table	{goiardi_schema}	{}	{}	2014-09-24 21:30:13.0901-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 12:09:28-07	Jeremy Bingham	jbingham@gmail.com
deploy	c4b32778f2911930f583ce15267aade320ac4dcd	sandboxes	goiardi_postgres	Create sandboxes table	{goiardi_schema}	{}	{}	2014-09-24 21:30:13.108413-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 12:14:48-07	Jeremy Bingham	jbingham@gmail.com
deploy	81003655b93b41359804027fc202788aa0ddd9a9	log_infos	goiardi_postgres	Create log_infos table	{clients,users,goiardi_schema}	{}	{}	2014-09-24 21:30:13.142359-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 12:19:10-07	Jeremy Bingham	jbingham@gmail.com
deploy	fce5b7aeed2ad742de1309d7841577cff19475a7	organizations	goiardi_postgres	Create organizations table	{}	{}	{}	2014-09-24 21:30:13.16193-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 12:46:28-07	Jeremy Bingham	jbingham@gmail.com
deploy	f2621482d1c130ea8fee15d09f966685409bf67c	file_checksums	goiardi_postgres	Create file checksums table	{}	{}	{}	2014-09-24 21:30:13.18021-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 12:49:19-07	Jeremy Bingham	jbingham@gmail.com
deploy	db1eb360cd5e6449a468ceb781d82b45dafb5c2d	reports	goiardi_postgres	Create reports table	{}	{}	{}	2014-09-24 21:30:13.203369-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 13:02:49-07	Jeremy Bingham	jbingham@gmail.com
deploy	c8b38382f7e5a18f36c621327f59205aa8aa9849	client_insert_duplicate	goiardi_postgres	Function to emulate insert ... on duplicate update for clients	{clients,goiardi_schema}	{}	{}	2014-09-24 21:30:13.221215-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 23:00:04-07	Jeremy Bingham	jbingham@gmail.com
deploy	30774a960a0efb6adfbb1d526b8cdb1a45c7d039	client_rename	goiardi_postgres	Function to rename clients	{clients,goiardi_schema}	{}	{}	2014-09-24 21:30:13.235098-07	Jeremy Bingham	jbingham@gmail.com	2014-05-30 10:22:50-07	Jeremy Bingham	jbingham@gmail.com
deploy	2d1fdc8128b0632e798df7346e76f122ed5915ec	user_insert_duplicate	goiardi_postgres	Function to emulate insert ... on duplicate update for clients	{users,goiardi_schema}	{}	{}	2014-09-24 21:30:13.249911-07	Jeremy Bingham	jbingham@gmail.com	2014-05-30 13:07:46-07	Jeremy Bingham	jbingham@gmail.com
deploy	f336c149ab32530c9c6ae4408c11558a635f39a1	user_rename	goiardi_postgres	Function to rename users	{users,goiardi_schema}	{}	{}	2014-09-24 21:30:13.264442-07	Jeremy Bingham	jbingham@gmail.com	2014-05-30 13:15:45-07	Jeremy Bingham	jbingham@gmail.com
deploy	841a7d554d44f9d0d0b8a1a5a9d0a06ce71a2453	cookbook_insert_update	goiardi_postgres	Cookbook insert/update	{cookbooks,goiardi_schema}	{}	{}	2014-09-24 21:30:13.281444-07	Jeremy Bingham	jbingham@gmail.com	2014-05-30 23:55:23-07	Jeremy Bingham	jbingham@gmail.com
deploy	085e2f6281914c9fa6521d59fea81f16c106b59f	cookbook_versions_insert_update	goiardi_postgres	Cookbook versions insert/update	{cookbook_versions,goiardi_schema}	{}	{}	2014-09-24 21:30:13.300569-07	Jeremy Bingham	jbingham@gmail.com	2014-05-30 23:56:05-07	Jeremy Bingham	jbingham@gmail.com
deploy	04bea39d649e4187d9579bd946fd60f760240d10	data_bag_insert_update	goiardi_postgres	Insert/update data bags	{data_bags,goiardi_schema}	{}	{}	2014-09-24 21:30:13.314896-07	Jeremy Bingham	jbingham@gmail.com	2014-05-31 23:25:44-07	Jeremy Bingham	jbingham@gmail.com
deploy	092885e8b5d94a9c1834bf309e02dc0f955ff053	environment_insert_update	goiardi_postgres	Insert/update environments	{environments,goiardi_schema}	{}	{}	2014-09-24 21:30:13.32947-07	Jeremy Bingham	jbingham@gmail.com	2014-06-01 12:55:34-07	Jeremy Bingham	jbingham@gmail.com
deploy	6d9587fa4275827c93ca9d7e0166ad1887b76cad	file_checksum_insert_ignore	goiardi_postgres	Insert ignore for file checksums	{file_checksums,goiardi_schema}	{}	{}	2014-09-24 21:30:13.344726-07	Jeremy Bingham	jbingham@gmail.com	2014-06-01 23:13:48-07	Jeremy Bingham	jbingham@gmail.com
deploy	82a95e5e6cbd8ba51fea33506e1edb2a12e37a92	node_insert_update	goiardi_postgres	Insert/update for nodes	{nodes,goiardi_schema}	{}	{}	2014-09-24 21:30:13.358592-07	Jeremy Bingham	jbingham@gmail.com	2014-06-01 23:25:20-07	Jeremy Bingham	jbingham@gmail.com
deploy	d052a8267a6512581e5cab1f89a2456f279727b9	report_insert_update	goiardi_postgres	Insert/update for reports	{reports,goiardi_schema}	{}	{}	2014-09-24 21:30:13.372684-07	Jeremy Bingham	jbingham@gmail.com	2014-06-02 10:10:25-07	Jeremy Bingham	jbingham@gmail.com
deploy	acf76029633d50febbec7c4763b7173078eddaf7	role_insert_update	goiardi_postgres	Insert/update for roles	{roles,goiardi_schema}	{}	{}	2014-09-24 21:30:13.387215-07	Jeremy Bingham	jbingham@gmail.com	2014-06-02 10:27:32-07	Jeremy Bingham	jbingham@gmail.com
deploy	b8ef36df686397ecb0fe67eb097e84aa0d78ac6b	sandbox_insert_update	goiardi_postgres	Insert/update for sandboxes	{sandboxes,goiardi_schema}	{}	{}	2014-09-24 21:30:13.401409-07	Jeremy Bingham	jbingham@gmail.com	2014-06-02 10:34:39-07	Jeremy Bingham	jbingham@gmail.com
deploy	93dbbda50a25da0a586e89ccee8fcfa2ddcb7c64	data_bag_item_insert	goiardi_postgres	Insert for data bag items	{data_bag_items,data_bags,goiardi_schema}	{}	{@v0.6.0}	2014-09-24 21:30:13.417742-07	Jeremy Bingham	jbingham@gmail.com	2014-06-02 14:03:22-07	Jeremy Bingham	jbingham@gmail.com
deploy	c80c561c22f6e139165cdb338c7ce6fff8ff268d	bytea_to_json	goiardi_postgres	Change most postgres bytea fields to json, because in this peculiar case json is way faster than gob	{}	{}	{}	2014-09-24 21:30:13.465841-07	Jeremy Bingham	jbingham@gmail.com	2014-07-20 02:41:22-07	Jeremy Bingham	jbingham@gmail.com
deploy	9966894e0fc0da573243f6a3c0fc1432a2b63043	joined_cookbkook_version	goiardi_postgres	a convenient view for joined versions for cookbook versions, adapted from erchef's joined_cookbook_version	{}	{}	{@v0.7.0}	2014-09-24 21:30:13.489334-07	Jeremy Bingham	jbingham@gmail.com	2014-07-20 03:21:28-07	Jeremy Bingham	jbingham@gmail.com
deploy	163ba4a496b9b4210d335e0e4ea5368a9ea8626c	node_statuses	goiardi_postgres	Create node_status table for node statuses	{nodes}	{}	{}	2014-09-24 21:30:13.508688-07	Jeremy Bingham	jbingham@gmail.com	2014-07-10 23:01:54-07	Jeremy Bingham	jeremy@terqa.local
deploy	8bb822f391b499585cfb2fc7248be469b0200682	node_status_insert	goiardi_postgres	insert function for node_statuses	{node_statuses}	{}	{}	2014-09-24 21:30:13.52333-07	Jeremy Bingham	jbingham@gmail.com	2014-07-11 00:01:31-07	Jeremy Bingham	jeremy@terqa.local
deploy	7c429aac08527adc774767584201f668408b04a6	add_down_column_nodes	goiardi_postgres	Add is_down column to the nodes table	{nodes}	{}	{}	2014-09-24 21:30:13.54375-07	Jeremy Bingham	jbingham@gmail.com	2014-07-15 20:18:05-07	Jeremy Bingham	jbingham@gmail.com
deploy	82bcace325dbdc905eb6e677f800d14a0506a216	shovey	goiardi_postgres	add shovey tables	{}	{}	{}	2014-09-24 21:30:13.576541-07	Jeremy Bingham	jbingham@gmail.com	2014-07-15 22:07:12-07	Jeremy Bingham	jeremy@terqa.local
deploy	62046d2fb96bbaedce2406252d312766452551c0	node_latest_statuses	goiardi_postgres	Add a view to easily get nodes by their latest status	{node_statuses}	{}	{}	2014-09-24 21:30:13.599649-07	Jeremy Bingham	jbingham@gmail.com	2014-07-26 13:32:02-07	Jeremy Bingham	jbingham@gmail.com
deploy	68f90e1fd2aac6a117d7697626741a02b8d0ebbe	shovey_insert_update	goiardi_postgres	insert/update functions for shovey	{shovey}	{}	{@v0.8.0}	2014-09-24 21:30:13.618819-07	Jeremy Bingham	jbingham@gmail.com	2014-08-27 00:46:20-07	Jeremy Bingham	jbingham@gmail.com
revert	68f90e1fd2aac6a117d7697626741a02b8d0ebbe	shovey_insert_update	goiardi_postgres	insert/update functions for shovey	{shovey}	{}	{@v0.8.0}	2016-10-24 01:09:31.088-07	Jeremy Bingham	jeremy@goiardi.gl	2014-08-27 00:46:20-07	Jeremy Bingham	jbingham@gmail.com
revert	62046d2fb96bbaedce2406252d312766452551c0	node_latest_statuses	goiardi_postgres	Add a view to easily get nodes by their latest status	{node_statuses}	{}	{}	2016-10-24 01:09:31.115754-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-26 13:32:02-07	Jeremy Bingham	jbingham@gmail.com
revert	82bcace325dbdc905eb6e677f800d14a0506a216	shovey	goiardi_postgres	add shovey tables	{}	{}	{}	2016-10-24 01:09:31.165548-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-15 22:07:12-07	Jeremy Bingham	jeremy@terqa.local
revert	7c429aac08527adc774767584201f668408b04a6	add_down_column_nodes	goiardi_postgres	Add is_down column to the nodes table	{nodes}	{}	{}	2016-10-24 01:09:31.184977-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-15 20:18:05-07	Jeremy Bingham	jbingham@gmail.com
revert	8bb822f391b499585cfb2fc7248be469b0200682	node_status_insert	goiardi_postgres	insert function for node_statuses	{node_statuses}	{}	{}	2016-10-24 01:09:31.202384-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-11 00:01:31-07	Jeremy Bingham	jeremy@terqa.local
revert	163ba4a496b9b4210d335e0e4ea5368a9ea8626c	node_statuses	goiardi_postgres	Create node_status table for node statuses	{nodes}	{}	{}	2016-10-24 01:09:31.225444-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-10 23:01:54-07	Jeremy Bingham	jeremy@terqa.local
revert	9966894e0fc0da573243f6a3c0fc1432a2b63043	joined_cookbkook_version	goiardi_postgres	a convenient view for joined versions for cookbook versions, adapted from erchef's joined_cookbook_version	{}	{}	{@v0.7.0}	2016-10-24 01:09:31.243109-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-20 03:21:28-07	Jeremy Bingham	jbingham@gmail.com
revert	c80c561c22f6e139165cdb338c7ce6fff8ff268d	bytea_to_json	goiardi_postgres	Change most postgres bytea fields to json, because in this peculiar case json is way faster than gob	{}	{}	{}	2016-10-24 01:09:31.351998-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-20 02:41:22-07	Jeremy Bingham	jbingham@gmail.com
revert	93dbbda50a25da0a586e89ccee8fcfa2ddcb7c64	data_bag_item_insert	goiardi_postgres	Insert for data bag items	{data_bag_items,data_bags,goiardi_schema}	{}	{@v0.6.0}	2016-10-24 01:09:31.367718-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-02 14:03:22-07	Jeremy Bingham	jbingham@gmail.com
revert	b8ef36df686397ecb0fe67eb097e84aa0d78ac6b	sandbox_insert_update	goiardi_postgres	Insert/update for sandboxes	{sandboxes,goiardi_schema}	{}	{}	2016-10-24 01:09:31.382719-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-02 10:34:39-07	Jeremy Bingham	jbingham@gmail.com
revert	acf76029633d50febbec7c4763b7173078eddaf7	role_insert_update	goiardi_postgres	Insert/update for roles	{roles,goiardi_schema}	{}	{}	2016-10-24 01:09:31.398701-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-02 10:27:32-07	Jeremy Bingham	jbingham@gmail.com
revert	d052a8267a6512581e5cab1f89a2456f279727b9	report_insert_update	goiardi_postgres	Insert/update for reports	{reports,goiardi_schema}	{}	{}	2016-10-24 01:09:31.414228-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-02 10:10:25-07	Jeremy Bingham	jbingham@gmail.com
revert	82a95e5e6cbd8ba51fea33506e1edb2a12e37a92	node_insert_update	goiardi_postgres	Insert/update for nodes	{nodes,goiardi_schema}	{}	{}	2016-10-24 01:09:31.429988-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-01 23:25:20-07	Jeremy Bingham	jbingham@gmail.com
revert	6d9587fa4275827c93ca9d7e0166ad1887b76cad	file_checksum_insert_ignore	goiardi_postgres	Insert ignore for file checksums	{file_checksums,goiardi_schema}	{}	{}	2016-10-24 01:09:31.445107-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-01 23:13:48-07	Jeremy Bingham	jbingham@gmail.com
revert	092885e8b5d94a9c1834bf309e02dc0f955ff053	environment_insert_update	goiardi_postgres	Insert/update environments	{environments,goiardi_schema}	{}	{}	2016-10-24 01:09:31.46088-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-01 12:55:34-07	Jeremy Bingham	jbingham@gmail.com
revert	04bea39d649e4187d9579bd946fd60f760240d10	data_bag_insert_update	goiardi_postgres	Insert/update data bags	{data_bags,goiardi_schema}	{}	{}	2016-10-24 01:09:31.476739-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-31 23:25:44-07	Jeremy Bingham	jbingham@gmail.com
revert	085e2f6281914c9fa6521d59fea81f16c106b59f	cookbook_versions_insert_update	goiardi_postgres	Cookbook versions insert/update	{cookbook_versions,goiardi_schema}	{}	{}	2016-10-24 01:09:31.492979-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-30 23:56:05-07	Jeremy Bingham	jbingham@gmail.com
revert	841a7d554d44f9d0d0b8a1a5a9d0a06ce71a2453	cookbook_insert_update	goiardi_postgres	Cookbook insert/update	{cookbooks,goiardi_schema}	{}	{}	2016-10-24 01:09:31.508095-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-30 23:55:23-07	Jeremy Bingham	jbingham@gmail.com
revert	f336c149ab32530c9c6ae4408c11558a635f39a1	user_rename	goiardi_postgres	Function to rename users	{users,goiardi_schema}	{}	{}	2016-10-24 01:09:31.524334-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-30 13:15:45-07	Jeremy Bingham	jbingham@gmail.com
revert	2d1fdc8128b0632e798df7346e76f122ed5915ec	user_insert_duplicate	goiardi_postgres	Function to emulate insert ... on duplicate update for clients	{users,goiardi_schema}	{}	{}	2016-10-24 01:09:31.540773-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-30 13:07:46-07	Jeremy Bingham	jbingham@gmail.com
revert	30774a960a0efb6adfbb1d526b8cdb1a45c7d039	client_rename	goiardi_postgres	Function to rename clients	{clients,goiardi_schema}	{}	{}	2016-10-24 01:09:31.555915-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-30 10:22:50-07	Jeremy Bingham	jbingham@gmail.com
revert	c8b38382f7e5a18f36c621327f59205aa8aa9849	client_insert_duplicate	goiardi_postgres	Function to emulate insert ... on duplicate update for clients	{clients,goiardi_schema}	{}	{}	2016-10-24 01:09:31.57174-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 23:00:04-07	Jeremy Bingham	jbingham@gmail.com
revert	db1eb360cd5e6449a468ceb781d82b45dafb5c2d	reports	goiardi_postgres	Create reports table	{}	{}	{}	2016-10-24 01:09:31.591656-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 13:02:49-07	Jeremy Bingham	jbingham@gmail.com
revert	f2621482d1c130ea8fee15d09f966685409bf67c	file_checksums	goiardi_postgres	Create file checksums table	{}	{}	{}	2016-10-24 01:09:31.612299-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 12:49:19-07	Jeremy Bingham	jbingham@gmail.com
revert	fce5b7aeed2ad742de1309d7841577cff19475a7	organizations	goiardi_postgres	Create organizations table	{}	{}	{}	2016-10-24 01:09:31.635748-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 12:46:28-07	Jeremy Bingham	jbingham@gmail.com
revert	81003655b93b41359804027fc202788aa0ddd9a9	log_infos	goiardi_postgres	Create log_infos table	{clients,users,goiardi_schema}	{}	{}	2016-10-24 01:09:31.66493-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 12:19:10-07	Jeremy Bingham	jbingham@gmail.com
revert	c4b32778f2911930f583ce15267aade320ac4dcd	sandboxes	goiardi_postgres	Create sandboxes table	{goiardi_schema}	{}	{}	2016-10-24 01:09:31.682853-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 12:14:48-07	Jeremy Bingham	jbingham@gmail.com
revert	6a4489d9436ba1541d272700b303410cc906b08f	roles	goiardi_postgres	Create roles table	{goiardi_schema}	{}	{}	2016-10-24 01:09:31.702078-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 12:09:28-07	Jeremy Bingham	jbingham@gmail.com
revert	feddf91b62caed36c790988bd29222591980433b	data_bag_items	goiardi_postgres	Create data bag items table	{data_bags,goiardi_schema}	{}	{}	2016-10-24 01:09:31.720907-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 12:02:31-07	Jeremy Bingham	jbingham@gmail.com
revert	85483913f96710c1267c6abacb6568cef9327f15	data_bags	goiardi_postgres	Create cookbook data bags table	{goiardi_schema}	{}	{}	2016-10-24 01:09:31.7437-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 11:42:04-07	Jeremy Bingham	jbingham@gmail.com
revert	f529038064a0259bdecbdab1f9f665e17ddb6136	cookbook_versions	goiardi_postgres	Create cookbook versions table	{cookbooks,goiardi_schema}	{}	{}	2016-10-24 01:09:31.762687-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 11:31:34-07	Jeremy Bingham	jbingham@gmail.com
revert	138bc49d92c0bbb024cea41532a656f2d7f9b072	cookbooks	goiardi_postgres	Create cookbook  table	{goiardi_schema}	{}	{}	2016-10-24 01:09:31.785192-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 11:27:27-07	Jeremy Bingham	jbingham@gmail.com
revert	bb82d8869ffca8ba3d03a1502c50dbb3eee7a2e0	users	goiardi_postgres	Create user table	{goiardi_schema}	{}	{}	2016-10-24 01:09:31.808703-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 11:15:02-07	Jeremy Bingham	jbingham@gmail.com
revert	faa3571aa479de60f25785e707433b304ba3d2c7	clients	goiardi_postgres	Create client table	{goiardi_schema}	{}	{}	2016-10-24 01:09:31.83197-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 11:05:33-07	Jeremy Bingham	jbingham@gmail.com
revert	911c456769628c817340ee77fc8d2b7c1d697782	nodes	goiardi_postgres	Create node table	{goiardi_schema}	{}	{}	2016-10-24 01:09:31.850634-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 10:37:46-07	Jeremy Bingham	jbingham@gmail.com
revert	367c28670efddf25455b9fd33c23a5a278b08bb4	environments	goiardi_postgres	Environments for postgres	{goiardi_schema}	{}	{}	2016-10-24 01:09:31.869442-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 00:40:11-07	Jeremy Bingham	jbingham@gmail.com
revert	c89b0e25c808b327036c88e6c9750c7526314c86	goiardi_schema	goiardi_postgres	Add schema for goiardi-postgres	{}	{}	{}	2016-10-24 01:09:31.885749-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-27 14:09:07-07	Jeremy Bingham	jbingham@gmail.com
deploy	c89b0e25c808b327036c88e6c9750c7526314c86	goiardi_schema	goiardi_postgres	Add schema for goiardi-postgres	{}	{}	{}	2016-10-24 01:09:39.168855-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-27 14:09:07-07	Jeremy Bingham	jbingham@gmail.com
deploy	367c28670efddf25455b9fd33c23a5a278b08bb4	environments	goiardi_postgres	Environments for postgres	{goiardi_schema}	{}	{}	2016-10-24 01:09:39.198049-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 00:40:11-07	Jeremy Bingham	jbingham@gmail.com
deploy	911c456769628c817340ee77fc8d2b7c1d697782	nodes	goiardi_postgres	Create node table	{goiardi_schema}	{}	{}	2016-10-24 01:09:39.222956-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 10:37:46-07	Jeremy Bingham	jbingham@gmail.com
deploy	faa3571aa479de60f25785e707433b304ba3d2c7	clients	goiardi_postgres	Create client table	{goiardi_schema}	{}	{}	2016-10-24 01:09:39.247841-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 11:05:33-07	Jeremy Bingham	jbingham@gmail.com
deploy	bb82d8869ffca8ba3d03a1502c50dbb3eee7a2e0	users	goiardi_postgres	Create user table	{goiardi_schema}	{}	{}	2016-10-24 01:09:39.269937-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 11:15:02-07	Jeremy Bingham	jbingham@gmail.com
deploy	138bc49d92c0bbb024cea41532a656f2d7f9b072	cookbooks	goiardi_postgres	Create cookbook  table	{goiardi_schema}	{}	{}	2016-10-24 01:09:39.292306-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 11:27:27-07	Jeremy Bingham	jbingham@gmail.com
deploy	f529038064a0259bdecbdab1f9f665e17ddb6136	cookbook_versions	goiardi_postgres	Create cookbook versions table	{cookbooks,goiardi_schema}	{}	{}	2016-10-24 01:09:39.323669-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 11:31:34-07	Jeremy Bingham	jbingham@gmail.com
deploy	85483913f96710c1267c6abacb6568cef9327f15	data_bags	goiardi_postgres	Create cookbook data bags table	{goiardi_schema}	{}	{}	2016-10-24 01:09:39.346622-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 11:42:04-07	Jeremy Bingham	jbingham@gmail.com
deploy	feddf91b62caed36c790988bd29222591980433b	data_bag_items	goiardi_postgres	Create data bag items table	{data_bags,goiardi_schema}	{}	{}	2016-10-24 01:09:39.37133-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 12:02:31-07	Jeremy Bingham	jbingham@gmail.com
deploy	6a4489d9436ba1541d272700b303410cc906b08f	roles	goiardi_postgres	Create roles table	{goiardi_schema}	{}	{}	2016-10-24 01:09:39.393811-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 12:09:28-07	Jeremy Bingham	jbingham@gmail.com
deploy	c4b32778f2911930f583ce15267aade320ac4dcd	sandboxes	goiardi_postgres	Create sandboxes table	{goiardi_schema}	{}	{}	2016-10-24 01:09:39.415652-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 12:14:48-07	Jeremy Bingham	jbingham@gmail.com
deploy	81003655b93b41359804027fc202788aa0ddd9a9	log_infos	goiardi_postgres	Create log_infos table	{clients,users,goiardi_schema}	{}	{}	2016-10-24 01:09:39.447295-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 12:19:10-07	Jeremy Bingham	jbingham@gmail.com
deploy	fce5b7aeed2ad742de1309d7841577cff19475a7	organizations	goiardi_postgres	Create organizations table	{}	{}	{}	2016-10-24 01:09:39.468147-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 12:46:28-07	Jeremy Bingham	jbingham@gmail.com
deploy	f2621482d1c130ea8fee15d09f966685409bf67c	file_checksums	goiardi_postgres	Create file checksums table	{}	{}	{}	2016-10-24 01:09:39.494984-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 12:49:19-07	Jeremy Bingham	jbingham@gmail.com
deploy	db1eb360cd5e6449a468ceb781d82b45dafb5c2d	reports	goiardi_postgres	Create reports table	{}	{}	{}	2016-10-24 01:09:39.520058-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 13:02:49-07	Jeremy Bingham	jbingham@gmail.com
deploy	c8b38382f7e5a18f36c621327f59205aa8aa9849	client_insert_duplicate	goiardi_postgres	Function to emulate insert ... on duplicate update for clients	{clients,goiardi_schema}	{}	{}	2016-10-24 01:09:39.539611-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 23:00:04-07	Jeremy Bingham	jbingham@gmail.com
deploy	30774a960a0efb6adfbb1d526b8cdb1a45c7d039	client_rename	goiardi_postgres	Function to rename clients	{clients,goiardi_schema}	{}	{}	2016-10-24 01:09:39.559639-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-30 10:22:50-07	Jeremy Bingham	jbingham@gmail.com
deploy	2d1fdc8128b0632e798df7346e76f122ed5915ec	user_insert_duplicate	goiardi_postgres	Function to emulate insert ... on duplicate update for clients	{users,goiardi_schema}	{}	{}	2016-10-24 01:09:39.578255-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-30 13:07:46-07	Jeremy Bingham	jbingham@gmail.com
deploy	f336c149ab32530c9c6ae4408c11558a635f39a1	user_rename	goiardi_postgres	Function to rename users	{users,goiardi_schema}	{}	{}	2016-10-24 01:09:39.596874-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-30 13:15:45-07	Jeremy Bingham	jbingham@gmail.com
deploy	841a7d554d44f9d0d0b8a1a5a9d0a06ce71a2453	cookbook_insert_update	goiardi_postgres	Cookbook insert/update	{cookbooks,goiardi_schema}	{}	{}	2016-10-24 01:09:39.614035-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-30 23:55:23-07	Jeremy Bingham	jbingham@gmail.com
deploy	085e2f6281914c9fa6521d59fea81f16c106b59f	cookbook_versions_insert_update	goiardi_postgres	Cookbook versions insert/update	{cookbook_versions,goiardi_schema}	{}	{}	2016-10-24 01:09:39.632293-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-30 23:56:05-07	Jeremy Bingham	jbingham@gmail.com
deploy	04bea39d649e4187d9579bd946fd60f760240d10	data_bag_insert_update	goiardi_postgres	Insert/update data bags	{data_bags,goiardi_schema}	{}	{}	2016-10-24 01:09:39.649631-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-31 23:25:44-07	Jeremy Bingham	jbingham@gmail.com
deploy	092885e8b5d94a9c1834bf309e02dc0f955ff053	environment_insert_update	goiardi_postgres	Insert/update environments	{environments,goiardi_schema}	{}	{}	2016-10-24 01:09:39.668155-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-01 12:55:34-07	Jeremy Bingham	jbingham@gmail.com
deploy	6d9587fa4275827c93ca9d7e0166ad1887b76cad	file_checksum_insert_ignore	goiardi_postgres	Insert ignore for file checksums	{file_checksums,goiardi_schema}	{}	{}	2016-10-24 01:09:39.690203-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-01 23:13:48-07	Jeremy Bingham	jbingham@gmail.com
deploy	82a95e5e6cbd8ba51fea33506e1edb2a12e37a92	node_insert_update	goiardi_postgres	Insert/update for nodes	{nodes,goiardi_schema}	{}	{}	2016-10-24 01:09:39.709134-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-01 23:25:20-07	Jeremy Bingham	jbingham@gmail.com
deploy	d052a8267a6512581e5cab1f89a2456f279727b9	report_insert_update	goiardi_postgres	Insert/update for reports	{reports,goiardi_schema}	{}	{}	2016-10-24 01:09:39.727523-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-02 10:10:25-07	Jeremy Bingham	jbingham@gmail.com
deploy	acf76029633d50febbec7c4763b7173078eddaf7	role_insert_update	goiardi_postgres	Insert/update for roles	{roles,goiardi_schema}	{}	{}	2016-10-24 01:09:39.744685-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-02 10:27:32-07	Jeremy Bingham	jbingham@gmail.com
deploy	b8ef36df686397ecb0fe67eb097e84aa0d78ac6b	sandbox_insert_update	goiardi_postgres	Insert/update for sandboxes	{sandboxes,goiardi_schema}	{}	{}	2016-10-24 01:09:39.763319-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-02 10:34:39-07	Jeremy Bingham	jbingham@gmail.com
deploy	93dbbda50a25da0a586e89ccee8fcfa2ddcb7c64	data_bag_item_insert	goiardi_postgres	Insert for data bag items	{data_bag_items,data_bags,goiardi_schema}	{}	{@v0.6.0}	2016-10-24 01:09:39.783801-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-02 14:03:22-07	Jeremy Bingham	jbingham@gmail.com
deploy	c80c561c22f6e139165cdb338c7ce6fff8ff268d	bytea_to_json	goiardi_postgres	Change most postgres bytea fields to json, because in this peculiar case json is way faster than gob	{}	{}	{}	2016-10-24 01:09:39.845109-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-20 02:41:22-07	Jeremy Bingham	jbingham@gmail.com
deploy	9966894e0fc0da573243f6a3c0fc1432a2b63043	joined_cookbkook_version	goiardi_postgres	a convenient view for joined versions for cookbook versions, adapted from erchef's joined_cookbook_version	{}	{}	{@v0.7.0}	2016-10-24 01:09:39.871233-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-20 03:21:28-07	Jeremy Bingham	jbingham@gmail.com
deploy	163ba4a496b9b4210d335e0e4ea5368a9ea8626c	node_statuses	goiardi_postgres	Create node_status table for node statuses	{nodes}	{}	{}	2016-10-24 01:09:39.89474-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-10 23:01:54-07	Jeremy Bingham	jeremy@terqa.local
deploy	8bb822f391b499585cfb2fc7248be469b0200682	node_status_insert	goiardi_postgres	insert function for node_statuses	{node_statuses}	{}	{}	2016-10-24 01:09:39.912111-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-11 00:01:31-07	Jeremy Bingham	jeremy@terqa.local
deploy	7c429aac08527adc774767584201f668408b04a6	add_down_column_nodes	goiardi_postgres	Add is_down column to the nodes table	{nodes}	{}	{}	2016-10-24 01:09:39.938356-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-15 20:18:05-07	Jeremy Bingham	jbingham@gmail.com
deploy	82bcace325dbdc905eb6e677f800d14a0506a216	shovey	goiardi_postgres	add shovey tables	{}	{}	{}	2016-10-24 01:09:39.97688-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-15 22:07:12-07	Jeremy Bingham	jeremy@terqa.local
deploy	62046d2fb96bbaedce2406252d312766452551c0	node_latest_statuses	goiardi_postgres	Add a view to easily get nodes by their latest status	{node_statuses}	{}	{}	2016-10-24 01:09:39.996132-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-26 13:32:02-07	Jeremy Bingham	jbingham@gmail.com
deploy	68f90e1fd2aac6a117d7697626741a02b8d0ebbe	shovey_insert_update	goiardi_postgres	insert/update functions for shovey	{shovey}	{}	{@v0.8.0}	2016-10-24 01:09:40.014944-07	Jeremy Bingham	jeremy@goiardi.gl	2014-08-27 00:46:20-07	Jeremy Bingham	jbingham@gmail.com
deploy	6f7aa2430e01cf33715828f1957d072cd5006d1c	ltree	goiardi_postgres	Add tables for ltree search for postgres	{}	{}	{}	2016-10-24 01:09:40.114088-07	Jeremy Bingham	jeremy@goiardi.gl	2015-04-10 23:21:26-07	Jeremy Bingham	jeremy@goiardi.gl
deploy	e7eb33b00d2fb6302e0c3979e9cac6fb80da377e	ltree_del_col	goiardi_postgres	procedure for deleting search collections	{}	{}	{}	2016-10-24 01:09:40.131941-07	Jeremy Bingham	jeremy@goiardi.gl	2015-04-12 12:33:15-07	Jeremy Bingham	jeremy@goiardi.gl
deploy	f49decbb15053ec5691093568450f642578ca460	ltree_del_item	goiardi_postgres	procedure for deleting search items	{}	{}	{@v0.10.0}	2016-10-24 01:09:40.150306-07	Jeremy Bingham	jeremy@goiardi.gl	2015-04-12 13:03:50-07	Jeremy Bingham	jeremy@goiardi.gl
deploy	d87c4dc108d4fa90942cc3bab8e619a58aef3d2d	jsonb	goiardi_postgres	Switch from json to jsonb columns. Will require using postgres 9.4+.	{}	{}	{}	2016-10-24 01:09:40.211033-07	Jeremy Bingham	jeremy@goiardi.gl	2016-09-09 01:17:31-07	Jeremy Bingham	jeremy@eridu.local
revert	d87c4dc108d4fa90942cc3bab8e619a58aef3d2d	jsonb	goiardi_postgres	Switch from json to jsonb columns. Will require using postgres 9.4+.	{}	{}	{}	2016-10-24 01:33:27.33499-07	Jeremy Bingham	jeremy@goiardi.gl	2016-09-09 01:17:31-07	Jeremy Bingham	jeremy@eridu.local
revert	f49decbb15053ec5691093568450f642578ca460	ltree_del_item	goiardi_postgres	procedure for deleting search items	{}	{}	{@v0.10.0}	2016-10-24 01:33:27.352346-07	Jeremy Bingham	jeremy@goiardi.gl	2015-04-12 13:03:50-07	Jeremy Bingham	jeremy@goiardi.gl
revert	e7eb33b00d2fb6302e0c3979e9cac6fb80da377e	ltree_del_col	goiardi_postgres	procedure for deleting search collections	{}	{}	{}	2016-10-24 01:33:27.368752-07	Jeremy Bingham	jeremy@goiardi.gl	2015-04-12 12:33:15-07	Jeremy Bingham	jeremy@goiardi.gl
revert	6f7aa2430e01cf33715828f1957d072cd5006d1c	ltree	goiardi_postgres	Add tables for ltree search for postgres	{}	{}	{}	2016-10-24 01:33:27.398888-07	Jeremy Bingham	jeremy@goiardi.gl	2015-04-10 23:21:26-07	Jeremy Bingham	jeremy@goiardi.gl
revert	68f90e1fd2aac6a117d7697626741a02b8d0ebbe	shovey_insert_update	goiardi_postgres	insert/update functions for shovey	{shovey}	{}	{@v0.8.0}	2016-10-24 01:33:27.414882-07	Jeremy Bingham	jeremy@goiardi.gl	2014-08-27 00:46:20-07	Jeremy Bingham	jbingham@gmail.com
revert	62046d2fb96bbaedce2406252d312766452551c0	node_latest_statuses	goiardi_postgres	Add a view to easily get nodes by their latest status	{node_statuses}	{}	{}	2016-10-24 01:33:27.431134-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-26 13:32:02-07	Jeremy Bingham	jbingham@gmail.com
revert	82bcace325dbdc905eb6e677f800d14a0506a216	shovey	goiardi_postgres	add shovey tables	{}	{}	{}	2016-10-24 01:33:27.521818-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-15 22:07:12-07	Jeremy Bingham	jeremy@terqa.local
revert	7c429aac08527adc774767584201f668408b04a6	add_down_column_nodes	goiardi_postgres	Add is_down column to the nodes table	{nodes}	{}	{}	2016-10-24 01:33:27.540905-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-15 20:18:05-07	Jeremy Bingham	jbingham@gmail.com
revert	8bb822f391b499585cfb2fc7248be469b0200682	node_status_insert	goiardi_postgres	insert function for node_statuses	{node_statuses}	{}	{}	2016-10-24 01:33:27.557552-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-11 00:01:31-07	Jeremy Bingham	jeremy@terqa.local
revert	163ba4a496b9b4210d335e0e4ea5368a9ea8626c	node_statuses	goiardi_postgres	Create node_status table for node statuses	{nodes}	{}	{}	2016-10-24 01:33:27.575297-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-10 23:01:54-07	Jeremy Bingham	jeremy@terqa.local
revert	9966894e0fc0da573243f6a3c0fc1432a2b63043	joined_cookbkook_version	goiardi_postgres	a convenient view for joined versions for cookbook versions, adapted from erchef's joined_cookbook_version	{}	{}	{@v0.7.0}	2016-10-24 01:33:27.591439-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-20 03:21:28-07	Jeremy Bingham	jbingham@gmail.com
revert	c80c561c22f6e139165cdb338c7ce6fff8ff268d	bytea_to_json	goiardi_postgres	Change most postgres bytea fields to json, because in this peculiar case json is way faster than gob	{}	{}	{}	2016-10-24 01:33:27.642199-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-20 02:41:22-07	Jeremy Bingham	jbingham@gmail.com
revert	93dbbda50a25da0a586e89ccee8fcfa2ddcb7c64	data_bag_item_insert	goiardi_postgres	Insert for data bag items	{data_bag_items,data_bags,goiardi_schema}	{}	{@v0.6.0}	2016-10-24 01:33:27.658462-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-02 14:03:22-07	Jeremy Bingham	jbingham@gmail.com
revert	b8ef36df686397ecb0fe67eb097e84aa0d78ac6b	sandbox_insert_update	goiardi_postgres	Insert/update for sandboxes	{sandboxes,goiardi_schema}	{}	{}	2016-10-24 01:33:27.6743-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-02 10:34:39-07	Jeremy Bingham	jbingham@gmail.com
revert	acf76029633d50febbec7c4763b7173078eddaf7	role_insert_update	goiardi_postgres	Insert/update for roles	{roles,goiardi_schema}	{}	{}	2016-10-24 01:33:27.690205-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-02 10:27:32-07	Jeremy Bingham	jbingham@gmail.com
revert	d052a8267a6512581e5cab1f89a2456f279727b9	report_insert_update	goiardi_postgres	Insert/update for reports	{reports,goiardi_schema}	{}	{}	2016-10-24 01:33:27.706268-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-02 10:10:25-07	Jeremy Bingham	jbingham@gmail.com
revert	82a95e5e6cbd8ba51fea33506e1edb2a12e37a92	node_insert_update	goiardi_postgres	Insert/update for nodes	{nodes,goiardi_schema}	{}	{}	2016-10-24 01:33:27.721594-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-01 23:25:20-07	Jeremy Bingham	jbingham@gmail.com
revert	6d9587fa4275827c93ca9d7e0166ad1887b76cad	file_checksum_insert_ignore	goiardi_postgres	Insert ignore for file checksums	{file_checksums,goiardi_schema}	{}	{}	2016-10-24 01:33:27.737764-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-01 23:13:48-07	Jeremy Bingham	jbingham@gmail.com
revert	092885e8b5d94a9c1834bf309e02dc0f955ff053	environment_insert_update	goiardi_postgres	Insert/update environments	{environments,goiardi_schema}	{}	{}	2016-10-24 01:33:27.752713-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-01 12:55:34-07	Jeremy Bingham	jbingham@gmail.com
revert	04bea39d649e4187d9579bd946fd60f760240d10	data_bag_insert_update	goiardi_postgres	Insert/update data bags	{data_bags,goiardi_schema}	{}	{}	2016-10-24 01:33:27.768956-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-31 23:25:44-07	Jeremy Bingham	jbingham@gmail.com
revert	085e2f6281914c9fa6521d59fea81f16c106b59f	cookbook_versions_insert_update	goiardi_postgres	Cookbook versions insert/update	{cookbook_versions,goiardi_schema}	{}	{}	2016-10-24 01:33:27.783997-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-30 23:56:05-07	Jeremy Bingham	jbingham@gmail.com
revert	841a7d554d44f9d0d0b8a1a5a9d0a06ce71a2453	cookbook_insert_update	goiardi_postgres	Cookbook insert/update	{cookbooks,goiardi_schema}	{}	{}	2016-10-24 01:33:27.80004-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-30 23:55:23-07	Jeremy Bingham	jbingham@gmail.com
revert	f336c149ab32530c9c6ae4408c11558a635f39a1	user_rename	goiardi_postgres	Function to rename users	{users,goiardi_schema}	{}	{}	2016-10-24 01:33:27.815147-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-30 13:15:45-07	Jeremy Bingham	jbingham@gmail.com
revert	2d1fdc8128b0632e798df7346e76f122ed5915ec	user_insert_duplicate	goiardi_postgres	Function to emulate insert ... on duplicate update for clients	{users,goiardi_schema}	{}	{}	2016-10-24 01:33:27.831373-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-30 13:07:46-07	Jeremy Bingham	jbingham@gmail.com
revert	30774a960a0efb6adfbb1d526b8cdb1a45c7d039	client_rename	goiardi_postgres	Function to rename clients	{clients,goiardi_schema}	{}	{}	2016-10-24 01:33:27.847257-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-30 10:22:50-07	Jeremy Bingham	jbingham@gmail.com
revert	c8b38382f7e5a18f36c621327f59205aa8aa9849	client_insert_duplicate	goiardi_postgres	Function to emulate insert ... on duplicate update for clients	{clients,goiardi_schema}	{}	{}	2016-10-24 01:33:27.863302-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 23:00:04-07	Jeremy Bingham	jbingham@gmail.com
revert	db1eb360cd5e6449a468ceb781d82b45dafb5c2d	reports	goiardi_postgres	Create reports table	{}	{}	{}	2016-10-24 01:33:27.881088-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 13:02:49-07	Jeremy Bingham	jbingham@gmail.com
revert	f2621482d1c130ea8fee15d09f966685409bf67c	file_checksums	goiardi_postgres	Create file checksums table	{}	{}	{}	2016-10-24 01:33:27.899227-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 12:49:19-07	Jeremy Bingham	jbingham@gmail.com
revert	fce5b7aeed2ad742de1309d7841577cff19475a7	organizations	goiardi_postgres	Create organizations table	{}	{}	{}	2016-10-24 01:33:27.915959-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 12:46:28-07	Jeremy Bingham	jbingham@gmail.com
revert	81003655b93b41359804027fc202788aa0ddd9a9	log_infos	goiardi_postgres	Create log_infos table	{clients,users,goiardi_schema}	{}	{}	2016-10-24 01:33:27.935047-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 12:19:10-07	Jeremy Bingham	jbingham@gmail.com
revert	c4b32778f2911930f583ce15267aade320ac4dcd	sandboxes	goiardi_postgres	Create sandboxes table	{goiardi_schema}	{}	{}	2016-10-24 01:33:27.952855-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 12:14:48-07	Jeremy Bingham	jbingham@gmail.com
revert	6a4489d9436ba1541d272700b303410cc906b08f	roles	goiardi_postgres	Create roles table	{goiardi_schema}	{}	{}	2016-10-24 01:33:27.970133-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 12:09:28-07	Jeremy Bingham	jbingham@gmail.com
revert	feddf91b62caed36c790988bd29222591980433b	data_bag_items	goiardi_postgres	Create data bag items table	{data_bags,goiardi_schema}	{}	{}	2016-10-24 01:33:27.987353-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 12:02:31-07	Jeremy Bingham	jbingham@gmail.com
revert	85483913f96710c1267c6abacb6568cef9327f15	data_bags	goiardi_postgres	Create cookbook data bags table	{goiardi_schema}	{}	{}	2016-10-24 01:33:28.005775-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 11:42:04-07	Jeremy Bingham	jbingham@gmail.com
revert	f529038064a0259bdecbdab1f9f665e17ddb6136	cookbook_versions	goiardi_postgres	Create cookbook versions table	{cookbooks,goiardi_schema}	{}	{}	2016-10-24 01:33:28.024788-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 11:31:34-07	Jeremy Bingham	jbingham@gmail.com
revert	138bc49d92c0bbb024cea41532a656f2d7f9b072	cookbooks	goiardi_postgres	Create cookbook  table	{goiardi_schema}	{}	{}	2016-10-24 01:33:28.044111-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 11:27:27-07	Jeremy Bingham	jbingham@gmail.com
revert	bb82d8869ffca8ba3d03a1502c50dbb3eee7a2e0	users	goiardi_postgres	Create user table	{goiardi_schema}	{}	{}	2016-10-24 01:33:28.062411-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 11:15:02-07	Jeremy Bingham	jbingham@gmail.com
revert	faa3571aa479de60f25785e707433b304ba3d2c7	clients	goiardi_postgres	Create client table	{goiardi_schema}	{}	{}	2016-10-24 01:33:28.079896-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 11:05:33-07	Jeremy Bingham	jbingham@gmail.com
revert	911c456769628c817340ee77fc8d2b7c1d697782	nodes	goiardi_postgres	Create node table	{goiardi_schema}	{}	{}	2016-10-24 01:33:28.097722-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 10:37:46-07	Jeremy Bingham	jbingham@gmail.com
revert	367c28670efddf25455b9fd33c23a5a278b08bb4	environments	goiardi_postgres	Environments for postgres	{goiardi_schema}	{}	{}	2016-10-24 01:33:28.114591-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 00:40:11-07	Jeremy Bingham	jbingham@gmail.com
revert	c89b0e25c808b327036c88e6c9750c7526314c86	goiardi_schema	goiardi_postgres	Add schema for goiardi-postgres	{}	{}	{}	2016-10-24 01:33:28.131102-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-27 14:09:07-07	Jeremy Bingham	jbingham@gmail.com
deploy	c89b0e25c808b327036c88e6c9750c7526314c86	goiardi_schema	goiardi_postgres	Add schema for goiardi-postgres	{}	{}	{}	2016-10-24 01:36:00.171511-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-27 14:09:07-07	Jeremy Bingham	jbingham@gmail.com
deploy	367c28670efddf25455b9fd33c23a5a278b08bb4	environments	goiardi_postgres	Environments for postgres	{goiardi_schema}	{}	{}	2016-10-24 01:36:00.194585-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 00:40:11-07	Jeremy Bingham	jbingham@gmail.com
deploy	911c456769628c817340ee77fc8d2b7c1d697782	nodes	goiardi_postgres	Create node table	{goiardi_schema}	{}	{}	2016-10-24 01:36:00.219183-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 10:37:46-07	Jeremy Bingham	jbingham@gmail.com
deploy	faa3571aa479de60f25785e707433b304ba3d2c7	clients	goiardi_postgres	Create client table	{goiardi_schema}	{}	{}	2016-10-24 01:36:00.242051-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 11:05:33-07	Jeremy Bingham	jbingham@gmail.com
deploy	bb82d8869ffca8ba3d03a1502c50dbb3eee7a2e0	users	goiardi_postgres	Create user table	{goiardi_schema}	{}	{}	2016-10-24 01:36:00.26513-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 11:15:02-07	Jeremy Bingham	jbingham@gmail.com
deploy	138bc49d92c0bbb024cea41532a656f2d7f9b072	cookbooks	goiardi_postgres	Create cookbook  table	{goiardi_schema}	{}	{}	2016-10-24 01:36:00.287013-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 11:27:27-07	Jeremy Bingham	jbingham@gmail.com
deploy	f529038064a0259bdecbdab1f9f665e17ddb6136	cookbook_versions	goiardi_postgres	Create cookbook versions table	{cookbooks,goiardi_schema}	{}	{}	2016-10-24 01:36:00.309594-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 11:31:34-07	Jeremy Bingham	jbingham@gmail.com
deploy	85483913f96710c1267c6abacb6568cef9327f15	data_bags	goiardi_postgres	Create cookbook data bags table	{goiardi_schema}	{}	{}	2016-10-24 01:36:00.330929-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 11:42:04-07	Jeremy Bingham	jbingham@gmail.com
deploy	feddf91b62caed36c790988bd29222591980433b	data_bag_items	goiardi_postgres	Create data bag items table	{data_bags,goiardi_schema}	{}	{}	2016-10-24 01:36:00.354029-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 12:02:31-07	Jeremy Bingham	jbingham@gmail.com
deploy	6a4489d9436ba1541d272700b303410cc906b08f	roles	goiardi_postgres	Create roles table	{goiardi_schema}	{}	{}	2016-10-24 01:36:00.377437-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 12:09:28-07	Jeremy Bingham	jbingham@gmail.com
deploy	c4b32778f2911930f583ce15267aade320ac4dcd	sandboxes	goiardi_postgres	Create sandboxes table	{goiardi_schema}	{}	{}	2016-10-24 01:36:00.398405-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 12:14:48-07	Jeremy Bingham	jbingham@gmail.com
deploy	81003655b93b41359804027fc202788aa0ddd9a9	log_infos	goiardi_postgres	Create log_infos table	{clients,users,goiardi_schema}	{}	{}	2016-10-24 01:36:00.425955-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 12:19:10-07	Jeremy Bingham	jbingham@gmail.com
deploy	fce5b7aeed2ad742de1309d7841577cff19475a7	organizations	goiardi_postgres	Create organizations table	{}	{}	{}	2016-10-24 01:36:00.448229-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 12:46:28-07	Jeremy Bingham	jbingham@gmail.com
deploy	f2621482d1c130ea8fee15d09f966685409bf67c	file_checksums	goiardi_postgres	Create file checksums table	{}	{}	{}	2016-10-24 01:36:00.469552-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 12:49:19-07	Jeremy Bingham	jbingham@gmail.com
deploy	db1eb360cd5e6449a468ceb781d82b45dafb5c2d	reports	goiardi_postgres	Create reports table	{}	{}	{}	2016-10-24 01:36:00.494757-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 13:02:49-07	Jeremy Bingham	jbingham@gmail.com
deploy	c8b38382f7e5a18f36c621327f59205aa8aa9849	client_insert_duplicate	goiardi_postgres	Function to emulate insert ... on duplicate update for clients	{clients,goiardi_schema}	{}	{}	2016-10-24 01:36:00.513896-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-29 23:00:04-07	Jeremy Bingham	jbingham@gmail.com
deploy	30774a960a0efb6adfbb1d526b8cdb1a45c7d039	client_rename	goiardi_postgres	Function to rename clients	{clients,goiardi_schema}	{}	{}	2016-10-24 01:36:00.532074-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-30 10:22:50-07	Jeremy Bingham	jbingham@gmail.com
deploy	2d1fdc8128b0632e798df7346e76f122ed5915ec	user_insert_duplicate	goiardi_postgres	Function to emulate insert ... on duplicate update for clients	{users,goiardi_schema}	{}	{}	2016-10-24 01:36:00.550093-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-30 13:07:46-07	Jeremy Bingham	jbingham@gmail.com
deploy	f336c149ab32530c9c6ae4408c11558a635f39a1	user_rename	goiardi_postgres	Function to rename users	{users,goiardi_schema}	{}	{}	2016-10-24 01:36:00.568148-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-30 13:15:45-07	Jeremy Bingham	jbingham@gmail.com
deploy	841a7d554d44f9d0d0b8a1a5a9d0a06ce71a2453	cookbook_insert_update	goiardi_postgres	Cookbook insert/update	{cookbooks,goiardi_schema}	{}	{}	2016-10-24 01:36:00.586043-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-30 23:55:23-07	Jeremy Bingham	jbingham@gmail.com
deploy	085e2f6281914c9fa6521d59fea81f16c106b59f	cookbook_versions_insert_update	goiardi_postgres	Cookbook versions insert/update	{cookbook_versions,goiardi_schema}	{}	{}	2016-10-24 01:36:00.604051-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-30 23:56:05-07	Jeremy Bingham	jbingham@gmail.com
deploy	04bea39d649e4187d9579bd946fd60f760240d10	data_bag_insert_update	goiardi_postgres	Insert/update data bags	{data_bags,goiardi_schema}	{}	{}	2016-10-24 01:36:00.62237-07	Jeremy Bingham	jeremy@goiardi.gl	2014-05-31 23:25:44-07	Jeremy Bingham	jbingham@gmail.com
deploy	092885e8b5d94a9c1834bf309e02dc0f955ff053	environment_insert_update	goiardi_postgres	Insert/update environments	{environments,goiardi_schema}	{}	{}	2016-10-24 01:36:00.641359-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-01 12:55:34-07	Jeremy Bingham	jbingham@gmail.com
deploy	6d9587fa4275827c93ca9d7e0166ad1887b76cad	file_checksum_insert_ignore	goiardi_postgres	Insert ignore for file checksums	{file_checksums,goiardi_schema}	{}	{}	2016-10-24 01:36:00.65909-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-01 23:13:48-07	Jeremy Bingham	jbingham@gmail.com
deploy	82a95e5e6cbd8ba51fea33506e1edb2a12e37a92	node_insert_update	goiardi_postgres	Insert/update for nodes	{nodes,goiardi_schema}	{}	{}	2016-10-24 01:36:00.67813-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-01 23:25:20-07	Jeremy Bingham	jbingham@gmail.com
deploy	d052a8267a6512581e5cab1f89a2456f279727b9	report_insert_update	goiardi_postgres	Insert/update for reports	{reports,goiardi_schema}	{}	{}	2016-10-24 01:36:00.695589-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-02 10:10:25-07	Jeremy Bingham	jbingham@gmail.com
deploy	acf76029633d50febbec7c4763b7173078eddaf7	role_insert_update	goiardi_postgres	Insert/update for roles	{roles,goiardi_schema}	{}	{}	2016-10-24 01:36:00.714295-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-02 10:27:32-07	Jeremy Bingham	jbingham@gmail.com
deploy	b8ef36df686397ecb0fe67eb097e84aa0d78ac6b	sandbox_insert_update	goiardi_postgres	Insert/update for sandboxes	{sandboxes,goiardi_schema}	{}	{}	2016-10-24 01:36:00.73281-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-02 10:34:39-07	Jeremy Bingham	jbingham@gmail.com
deploy	93dbbda50a25da0a586e89ccee8fcfa2ddcb7c64	data_bag_item_insert	goiardi_postgres	Insert for data bag items	{data_bag_items,data_bags,goiardi_schema}	{}	{@v0.6.0}	2016-10-24 01:36:00.751818-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-02 14:03:22-07	Jeremy Bingham	jbingham@gmail.com
deploy	c80c561c22f6e139165cdb338c7ce6fff8ff268d	bytea_to_json	goiardi_postgres	Change most postgres bytea fields to json, because in this peculiar case json is way faster than gob	{}	{}	{}	2016-10-24 01:36:00.808472-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-20 02:41:22-07	Jeremy Bingham	jbingham@gmail.com
deploy	9966894e0fc0da573243f6a3c0fc1432a2b63043	joined_cookbkook_version	goiardi_postgres	a convenient view for joined versions for cookbook versions, adapted from erchef's joined_cookbook_version	{}	{}	{@v0.7.0}	2016-10-24 01:36:00.827828-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-20 03:21:28-07	Jeremy Bingham	jbingham@gmail.com
deploy	163ba4a496b9b4210d335e0e4ea5368a9ea8626c	node_statuses	goiardi_postgres	Create node_status table for node statuses	{nodes}	{}	{}	2016-10-24 01:36:00.85042-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-10 23:01:54-07	Jeremy Bingham	jeremy@terqa.local
deploy	8bb822f391b499585cfb2fc7248be469b0200682	node_status_insert	goiardi_postgres	insert function for node_statuses	{node_statuses}	{}	{}	2016-10-24 01:36:00.868871-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-11 00:01:31-07	Jeremy Bingham	jeremy@terqa.local
deploy	7c429aac08527adc774767584201f668408b04a6	add_down_column_nodes	goiardi_postgres	Add is_down column to the nodes table	{nodes}	{}	{}	2016-10-24 01:36:00.892679-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-15 20:18:05-07	Jeremy Bingham	jbingham@gmail.com
deploy	82bcace325dbdc905eb6e677f800d14a0506a216	shovey	goiardi_postgres	add shovey tables	{}	{}	{}	2016-10-24 01:36:00.929792-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-15 22:07:12-07	Jeremy Bingham	jeremy@terqa.local
deploy	62046d2fb96bbaedce2406252d312766452551c0	node_latest_statuses	goiardi_postgres	Add a view to easily get nodes by their latest status	{node_statuses}	{}	{}	2016-10-24 01:36:00.948526-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-26 13:32:02-07	Jeremy Bingham	jbingham@gmail.com
deploy	68f90e1fd2aac6a117d7697626741a02b8d0ebbe	shovey_insert_update	goiardi_postgres	insert/update functions for shovey	{shovey}	{}	{@v0.8.0}	2016-10-24 01:36:00.968921-07	Jeremy Bingham	jeremy@goiardi.gl	2014-08-27 00:46:20-07	Jeremy Bingham	jbingham@gmail.com
deploy	6f7aa2430e01cf33715828f1957d072cd5006d1c	ltree	goiardi_postgres	Add tables for ltree search for postgres	{}	{}	{}	2016-10-24 01:36:01.021112-07	Jeremy Bingham	jeremy@goiardi.gl	2015-04-10 23:21:26-07	Jeremy Bingham	jeremy@goiardi.gl
deploy	e7eb33b00d2fb6302e0c3979e9cac6fb80da377e	ltree_del_col	goiardi_postgres	procedure for deleting search collections	{}	{}	{}	2016-10-24 01:36:01.039484-07	Jeremy Bingham	jeremy@goiardi.gl	2015-04-12 12:33:15-07	Jeremy Bingham	jeremy@goiardi.gl
deploy	f49decbb15053ec5691093568450f642578ca460	ltree_del_item	goiardi_postgres	procedure for deleting search items	{}	{}	{@v0.10.0}	2016-10-24 01:36:01.057822-07	Jeremy Bingham	jeremy@goiardi.gl	2015-04-12 13:03:50-07	Jeremy Bingham	jeremy@goiardi.gl
deploy	d87c4dc108d4fa90942cc3bab8e619a58aef3d2d	jsonb	goiardi_postgres	Switch from json to jsonb columns. Will require using postgres 9.4+.	{}	{}	{@v0.11.0}	2016-10-24 01:36:01.115742-07	Jeremy Bingham	jeremy@goiardi.gl	2016-09-09 01:17:31-07	Jeremy Bingham	jeremy@eridu.local
\.


--
-- Data for Name: projects; Type: TABLE DATA; Schema: sqitch; Owner: -
--

COPY projects (project, uri, created_at, creator_name, creator_email) FROM stdin;
goiardi_postgres	http://ctdk.github.com/goiardi/postgres-support	2014-09-24 21:30:12.879145-07	Jeremy Bingham	jbingham@gmail.com
\.


--
-- Data for Name: releases; Type: TABLE DATA; Schema: sqitch; Owner: -
--

COPY releases (version, installed_at, installer_name, installer_email) FROM stdin;
1	2016-10-24 01:09:26.837866-07	Jeremy Bingham	jeremy@goiardi.gl
1.10000002	2016-10-24 01:09:26.879489-07	Jeremy Bingham	jeremy@goiardi.gl
\.


--
-- Data for Name: tags; Type: TABLE DATA; Schema: sqitch; Owner: -
--

COPY tags (tag_id, tag, project, change_id, note, committed_at, committer_name, committer_email, planned_at, planner_name, planner_email) FROM stdin;
fd6ca4c1426a85718d19687591885a2c2a516952	@v0.6.0	goiardi_postgres	93dbbda50a25da0a586e89ccee8fcfa2ddcb7c64	Tag v0.6.0 for release	2016-10-24 01:36:00.750728-07	Jeremy Bingham	jeremy@goiardi.gl	2014-06-27 00:20:56-07	Jeremy Bingham	jbingham@gmail.com
10ec54c07a54a2138c04d471dd6d4a2ce25677b1	@v0.7.0	goiardi_postgres	9966894e0fc0da573243f6a3c0fc1432a2b63043	Tag 0.7.0 postgres schema	2016-10-24 01:36:00.826861-07	Jeremy Bingham	jeremy@goiardi.gl	2014-07-20 23:04:53-07	Jeremy Bingham	jeremy@terqa.local
644417084f02f0e8c6249f6ee0c9bf17b3a037b2	@v0.8.0	goiardi_postgres	68f90e1fd2aac6a117d7697626741a02b8d0ebbe	Tag v0.8.0	2016-10-24 01:36:00.967896-07	Jeremy Bingham	jeremy@goiardi.gl	2014-09-24 21:17:41-07	Jeremy Bingham	jbingham@gmail.com
970e1b9f6fecc093ca76bf75314076afadcdb5fd	@v0.10.0	goiardi_postgres	f49decbb15053ec5691093568450f642578ca460	Tag the 0.10.0 release.	2016-10-24 01:36:01.056738-07	Jeremy Bingham	jeremy@goiardi.gl	2015-07-23 00:21:08-07	Jeremy Bingham	jeremy@goiardi.gl
d8aefb7cd8b09c8fb3d48244847dcbebe7eeda3e	@v0.11.0	goiardi_postgres	d87c4dc108d4fa90942cc3bab8e619a58aef3d2d	tag the 0.11.0 release schema	2016-10-24 01:36:01.114549-07	Jeremy Bingham	jeremy@goiardi.gl	2016-10-24 01:35:53-07	Jeremy Bingham	jeremy@goiardi.gl
\.


SET search_path = goiardi, pg_catalog;

--
-- Name: clients_organization_id_name_key; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY clients
    ADD CONSTRAINT clients_organization_id_name_key UNIQUE (organization_id, name);


--
-- Name: clients_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY clients
    ADD CONSTRAINT clients_pkey PRIMARY KEY (id);


--
-- Name: cookbook_versions_cookbook_id_major_ver_minor_ver_patch_ver_key; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY cookbook_versions
    ADD CONSTRAINT cookbook_versions_cookbook_id_major_ver_minor_ver_patch_ver_key UNIQUE (cookbook_id, major_ver, minor_ver, patch_ver);


--
-- Name: cookbook_versions_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY cookbook_versions
    ADD CONSTRAINT cookbook_versions_pkey PRIMARY KEY (id);


--
-- Name: cookbooks_organization_id_name_key; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY cookbooks
    ADD CONSTRAINT cookbooks_organization_id_name_key UNIQUE (organization_id, name);


--
-- Name: cookbooks_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY cookbooks
    ADD CONSTRAINT cookbooks_pkey PRIMARY KEY (id);


--
-- Name: data_bag_items_data_bag_id_name_key; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY data_bag_items
    ADD CONSTRAINT data_bag_items_data_bag_id_name_key UNIQUE (data_bag_id, name);


--
-- Name: data_bag_items_data_bag_id_orig_name_key; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY data_bag_items
    ADD CONSTRAINT data_bag_items_data_bag_id_orig_name_key UNIQUE (data_bag_id, orig_name);


--
-- Name: data_bag_items_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY data_bag_items
    ADD CONSTRAINT data_bag_items_pkey PRIMARY KEY (id);


--
-- Name: data_bags_organization_id_name_key; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY data_bags
    ADD CONSTRAINT data_bags_organization_id_name_key UNIQUE (organization_id, name);


--
-- Name: data_bags_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY data_bags
    ADD CONSTRAINT data_bags_pkey PRIMARY KEY (id);


--
-- Name: environments_organization_id_name_key; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY environments
    ADD CONSTRAINT environments_organization_id_name_key UNIQUE (organization_id, name);


--
-- Name: environments_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY environments
    ADD CONSTRAINT environments_pkey PRIMARY KEY (id);


--
-- Name: file_checksums_organization_id_checksum_key; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY file_checksums
    ADD CONSTRAINT file_checksums_organization_id_checksum_key UNIQUE (organization_id, checksum);


--
-- Name: file_checksums_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY file_checksums
    ADD CONSTRAINT file_checksums_pkey PRIMARY KEY (id);


--
-- Name: log_infos_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY log_infos
    ADD CONSTRAINT log_infos_pkey PRIMARY KEY (id);


--
-- Name: node_statuses_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY node_statuses
    ADD CONSTRAINT node_statuses_pkey PRIMARY KEY (id);


--
-- Name: nodes_organization_id_name_key; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY nodes
    ADD CONSTRAINT nodes_organization_id_name_key UNIQUE (organization_id, name);


--
-- Name: nodes_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY nodes
    ADD CONSTRAINT nodes_pkey PRIMARY KEY (id);


--
-- Name: organizations_name_key; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY organizations
    ADD CONSTRAINT organizations_name_key UNIQUE (name);


--
-- Name: organizations_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY organizations
    ADD CONSTRAINT organizations_pkey PRIMARY KEY (id);


--
-- Name: reports_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY reports
    ADD CONSTRAINT reports_pkey PRIMARY KEY (id);


--
-- Name: reports_run_id_key; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY reports
    ADD CONSTRAINT reports_run_id_key UNIQUE (run_id);


--
-- Name: roles_organization_id_name_key; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY roles
    ADD CONSTRAINT roles_organization_id_name_key UNIQUE (organization_id, name);


--
-- Name: roles_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY roles
    ADD CONSTRAINT roles_pkey PRIMARY KEY (id);


--
-- Name: sandboxes_organization_id_sbox_id_key; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY sandboxes
    ADD CONSTRAINT sandboxes_organization_id_sbox_id_key UNIQUE (organization_id, sbox_id);


--
-- Name: sandboxes_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY sandboxes
    ADD CONSTRAINT sandboxes_pkey PRIMARY KEY (id);


--
-- Name: search_collections_organization_id_name_key; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY search_collections
    ADD CONSTRAINT search_collections_organization_id_name_key UNIQUE (organization_id, name);


--
-- Name: search_collections_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY search_collections
    ADD CONSTRAINT search_collections_pkey PRIMARY KEY (id);


--
-- Name: search_items_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY search_items
    ADD CONSTRAINT search_items_pkey PRIMARY KEY (id);


--
-- Name: shovey_run_streams_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY shovey_run_streams
    ADD CONSTRAINT shovey_run_streams_pkey PRIMARY KEY (id);


--
-- Name: shovey_run_streams_shovey_run_id_output_type_seq_key; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY shovey_run_streams
    ADD CONSTRAINT shovey_run_streams_shovey_run_id_output_type_seq_key UNIQUE (shovey_run_id, output_type, seq);


--
-- Name: shovey_runs_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY shovey_runs
    ADD CONSTRAINT shovey_runs_pkey PRIMARY KEY (id);


--
-- Name: shovey_runs_shovey_id_node_name_key; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY shovey_runs
    ADD CONSTRAINT shovey_runs_shovey_id_node_name_key UNIQUE (shovey_id, node_name);


--
-- Name: shoveys_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY shoveys
    ADD CONSTRAINT shoveys_pkey PRIMARY KEY (id);


--
-- Name: shoveys_run_id_key; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY shoveys
    ADD CONSTRAINT shoveys_run_id_key UNIQUE (run_id);


--
-- Name: users_email_key; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY users
    ADD CONSTRAINT users_email_key UNIQUE (email);


--
-- Name: users_name_key; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY users
    ADD CONSTRAINT users_name_key UNIQUE (name);


--
-- Name: users_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


SET search_path = sqitch, pg_catalog;

--
-- Name: changes_pkey; Type: CONSTRAINT; Schema: sqitch; Owner: -
--

ALTER TABLE ONLY changes
    ADD CONSTRAINT changes_pkey PRIMARY KEY (change_id);


--
-- Name: changes_project_script_hash_key; Type: CONSTRAINT; Schema: sqitch; Owner: -
--

ALTER TABLE ONLY changes
    ADD CONSTRAINT changes_project_script_hash_key UNIQUE (project, script_hash);


--
-- Name: dependencies_pkey; Type: CONSTRAINT; Schema: sqitch; Owner: -
--

ALTER TABLE ONLY dependencies
    ADD CONSTRAINT dependencies_pkey PRIMARY KEY (change_id, dependency);


--
-- Name: events_pkey; Type: CONSTRAINT; Schema: sqitch; Owner: -
--

ALTER TABLE ONLY events
    ADD CONSTRAINT events_pkey PRIMARY KEY (change_id, committed_at);


--
-- Name: projects_pkey; Type: CONSTRAINT; Schema: sqitch; Owner: -
--

ALTER TABLE ONLY projects
    ADD CONSTRAINT projects_pkey PRIMARY KEY (project);


--
-- Name: projects_uri_key; Type: CONSTRAINT; Schema: sqitch; Owner: -
--

ALTER TABLE ONLY projects
    ADD CONSTRAINT projects_uri_key UNIQUE (uri);


--
-- Name: releases_pkey; Type: CONSTRAINT; Schema: sqitch; Owner: -
--

ALTER TABLE ONLY releases
    ADD CONSTRAINT releases_pkey PRIMARY KEY (version);


--
-- Name: tags_pkey; Type: CONSTRAINT; Schema: sqitch; Owner: -
--

ALTER TABLE ONLY tags
    ADD CONSTRAINT tags_pkey PRIMARY KEY (tag_id);


--
-- Name: tags_project_tag_key; Type: CONSTRAINT; Schema: sqitch; Owner: -
--

ALTER TABLE ONLY tags
    ADD CONSTRAINT tags_project_tag_key UNIQUE (project, tag);


SET search_path = goiardi, pg_catalog;

--
-- Name: log_info_orgs; Type: INDEX; Schema: goiardi; Owner: -
--

CREATE INDEX log_info_orgs ON log_infos USING btree (organization_id);


--
-- Name: log_infos_action; Type: INDEX; Schema: goiardi; Owner: -
--

CREATE INDEX log_infos_action ON log_infos USING btree (action);


--
-- Name: log_infos_actor; Type: INDEX; Schema: goiardi; Owner: -
--

CREATE INDEX log_infos_actor ON log_infos USING btree (actor_id);


--
-- Name: log_infos_obj; Type: INDEX; Schema: goiardi; Owner: -
--

CREATE INDEX log_infos_obj ON log_infos USING btree (object_type, object_name);


--
-- Name: log_infos_time; Type: INDEX; Schema: goiardi; Owner: -
--

CREATE INDEX log_infos_time ON log_infos USING btree ("time");


--
-- Name: node_is_down; Type: INDEX; Schema: goiardi; Owner: -
--

CREATE INDEX node_is_down ON nodes USING btree (is_down);


--
-- Name: node_status_status; Type: INDEX; Schema: goiardi; Owner: -
--

CREATE INDEX node_status_status ON node_statuses USING btree (status);


--
-- Name: node_status_time; Type: INDEX; Schema: goiardi; Owner: -
--

CREATE INDEX node_status_time ON node_statuses USING btree (updated_at);


--
-- Name: nodes_chef_env; Type: INDEX; Schema: goiardi; Owner: -
--

CREATE INDEX nodes_chef_env ON nodes USING btree (chef_environment);


--
-- Name: report_node_organization; Type: INDEX; Schema: goiardi; Owner: -
--

CREATE INDEX report_node_organization ON reports USING btree (node_name, organization_id);


--
-- Name: report_organization_id; Type: INDEX; Schema: goiardi; Owner: -
--

CREATE INDEX report_organization_id ON reports USING btree (organization_id);


--
-- Name: search_btree_idx; Type: INDEX; Schema: goiardi; Owner: -
--

CREATE INDEX search_btree_idx ON search_items USING btree (path);


--
-- Name: search_col_name; Type: INDEX; Schema: goiardi; Owner: -
--

CREATE INDEX search_col_name ON search_collections USING btree (name);


--
-- Name: search_gist_idx; Type: INDEX; Schema: goiardi; Owner: -
--

CREATE INDEX search_gist_idx ON search_items USING gist (path);


--
-- Name: search_item_val_trgm; Type: INDEX; Schema: goiardi; Owner: -
--

CREATE INDEX search_item_val_trgm ON search_items USING gist (value gist_trgm_ops);


--
-- Name: search_multi_gist_idx; Type: INDEX; Schema: goiardi; Owner: -
--

CREATE INDEX search_multi_gist_idx ON search_items USING gist (path, value gist_trgm_ops);


--
-- Name: search_org_col; Type: INDEX; Schema: goiardi; Owner: -
--

CREATE INDEX search_org_col ON search_items USING btree (organization_id, search_collection_id);


--
-- Name: search_org_col_name; Type: INDEX; Schema: goiardi; Owner: -
--

CREATE INDEX search_org_col_name ON search_items USING btree (organization_id, search_collection_id, item_name);


--
-- Name: search_org_id; Type: INDEX; Schema: goiardi; Owner: -
--

CREATE INDEX search_org_id ON search_items USING btree (organization_id);


--
-- Name: search_val; Type: INDEX; Schema: goiardi; Owner: -
--

CREATE INDEX search_val ON search_items USING btree (value);


--
-- Name: shovey_organization_id; Type: INDEX; Schema: goiardi; Owner: -
--

CREATE INDEX shovey_organization_id ON shoveys USING btree (organization_id);


--
-- Name: shovey_organization_run_id; Type: INDEX; Schema: goiardi; Owner: -
--

CREATE INDEX shovey_organization_run_id ON shoveys USING btree (run_id, organization_id);


--
-- Name: shovey_run_node_name; Type: INDEX; Schema: goiardi; Owner: -
--

CREATE INDEX shovey_run_node_name ON shovey_runs USING btree (node_name);


--
-- Name: shovey_run_run_id; Type: INDEX; Schema: goiardi; Owner: -
--

CREATE INDEX shovey_run_run_id ON shovey_runs USING btree (shovey_uuid);


--
-- Name: shovey_run_status; Type: INDEX; Schema: goiardi; Owner: -
--

CREATE INDEX shovey_run_status ON shovey_runs USING btree (status);


--
-- Name: shovey_stream; Type: INDEX; Schema: goiardi; Owner: -
--

CREATE INDEX shovey_stream ON shovey_run_streams USING btree (shovey_run_id, output_type);


--
-- Name: shovey_uuid_node; Type: INDEX; Schema: goiardi; Owner: -
--

CREATE INDEX shovey_uuid_node ON shovey_runs USING btree (shovey_uuid, node_name);


--
-- Name: shoveys_status; Type: INDEX; Schema: goiardi; Owner: -
--

CREATE INDEX shoveys_status ON shoveys USING btree (status);


--
-- Name: insert_ignore; Type: RULE; Schema: goiardi; Owner: -
--

CREATE RULE insert_ignore AS
    ON INSERT TO file_checksums
   WHERE (EXISTS ( SELECT 1
           FROM file_checksums
          WHERE ((file_checksums.organization_id = new.organization_id) AND ((file_checksums.checksum)::text = (new.checksum)::text)))) DO INSTEAD NOTHING;


--
-- Name: cookbook_versions_cookbook_id_fkey; Type: FK CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY cookbook_versions
    ADD CONSTRAINT cookbook_versions_cookbook_id_fkey FOREIGN KEY (cookbook_id) REFERENCES cookbooks(id) ON DELETE RESTRICT;


--
-- Name: data_bag_items_data_bag_id_fkey; Type: FK CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY data_bag_items
    ADD CONSTRAINT data_bag_items_data_bag_id_fkey FOREIGN KEY (data_bag_id) REFERENCES data_bags(id) ON DELETE RESTRICT;


--
-- Name: node_statuses_node_id_fkey; Type: FK CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY node_statuses
    ADD CONSTRAINT node_statuses_node_id_fkey FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE;


--
-- Name: search_items_search_collection_id_fkey; Type: FK CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY search_items
    ADD CONSTRAINT search_items_search_collection_id_fkey FOREIGN KEY (search_collection_id) REFERENCES search_collections(id) ON DELETE RESTRICT;


--
-- Name: shovey_run_streams_shovey_run_id_fkey; Type: FK CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY shovey_run_streams
    ADD CONSTRAINT shovey_run_streams_shovey_run_id_fkey FOREIGN KEY (shovey_run_id) REFERENCES shovey_runs(id) ON DELETE RESTRICT;


--
-- Name: shovey_runs_shovey_id_fkey; Type: FK CONSTRAINT; Schema: goiardi; Owner: -
--

ALTER TABLE ONLY shovey_runs
    ADD CONSTRAINT shovey_runs_shovey_id_fkey FOREIGN KEY (shovey_id) REFERENCES shoveys(id) ON DELETE RESTRICT;


SET search_path = sqitch, pg_catalog;

--
-- Name: changes_project_fkey; Type: FK CONSTRAINT; Schema: sqitch; Owner: -
--

ALTER TABLE ONLY changes
    ADD CONSTRAINT changes_project_fkey FOREIGN KEY (project) REFERENCES projects(project) ON UPDATE CASCADE;


--
-- Name: dependencies_change_id_fkey; Type: FK CONSTRAINT; Schema: sqitch; Owner: -
--

ALTER TABLE ONLY dependencies
    ADD CONSTRAINT dependencies_change_id_fkey FOREIGN KEY (change_id) REFERENCES changes(change_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: dependencies_dependency_id_fkey; Type: FK CONSTRAINT; Schema: sqitch; Owner: -
--

ALTER TABLE ONLY dependencies
    ADD CONSTRAINT dependencies_dependency_id_fkey FOREIGN KEY (dependency_id) REFERENCES changes(change_id) ON UPDATE CASCADE;


--
-- Name: events_project_fkey; Type: FK CONSTRAINT; Schema: sqitch; Owner: -
--

ALTER TABLE ONLY events
    ADD CONSTRAINT events_project_fkey FOREIGN KEY (project) REFERENCES projects(project) ON UPDATE CASCADE;


--
-- Name: tags_change_id_fkey; Type: FK CONSTRAINT; Schema: sqitch; Owner: -
--

ALTER TABLE ONLY tags
    ADD CONSTRAINT tags_change_id_fkey FOREIGN KEY (change_id) REFERENCES changes(change_id) ON UPDATE CASCADE;


--
-- Name: tags_project_fkey; Type: FK CONSTRAINT; Schema: sqitch; Owner: -
--

ALTER TABLE ONLY tags
    ADD CONSTRAINT tags_project_fkey FOREIGN KEY (project) REFERENCES projects(project) ON UPDATE CASCADE;


--
-- PostgreSQL database dump complete
--

