--
-- PostgreSQL database dump
--

SET statement_timeout = 0;
SET lock_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;

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
-- Name: insert_dbi(text, text, text, bigint, json); Type: FUNCTION; Schema: goiardi; Owner: -
--

CREATE FUNCTION insert_dbi(m_data_bag_name text, m_name text, m_orig_name text, m_dbag_id bigint, m_raw_data json) RETURNS bigint
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
-- Name: merge_cookbook_versions(bigint, boolean, json, json, json, json, json, json, json, json, json, json, bigint, bigint, bigint); Type: FUNCTION; Schema: goiardi; Owner: -
--

CREATE FUNCTION merge_cookbook_versions(c_id bigint, is_frozen boolean, defb json, libb json, attb json, recb json, prob json, resb json, temb json, roob json, filb json, metb json, maj bigint, min bigint, patch bigint) RETURNS bigint
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
-- Name: merge_environments(text, text, json, json, json); Type: FUNCTION; Schema: goiardi; Owner: -
--

CREATE FUNCTION merge_environments(m_name text, m_description text, m_default_attr json, m_override_attr json, m_cookbook_vers json) RETURNS void
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
-- Name: merge_nodes(text, text, json, json, json, json, json); Type: FUNCTION; Schema: goiardi; Owner: -
--

CREATE FUNCTION merge_nodes(m_name text, m_chef_environment text, m_run_list json, m_automatic_attr json, m_normal_attr json, m_default_attr json, m_override_attr json) RETURNS void
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
-- Name: merge_reports(uuid, text, timestamp with time zone, timestamp with time zone, integer, report_status, text, json, json); Type: FUNCTION; Schema: goiardi; Owner: -
--

CREATE FUNCTION merge_reports(m_run_id uuid, m_node_name text, m_start_time timestamp with time zone, m_end_time timestamp with time zone, m_total_res_count integer, m_status report_status, m_run_list text, m_resources json, m_data json) RETURNS void
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
-- Name: merge_roles(text, text, json, json, json, json); Type: FUNCTION; Schema: goiardi; Owner: -
--

CREATE FUNCTION merge_roles(m_name text, m_description text, m_run_list json, m_env_run_lists json, m_default_attr json, m_override_attr json) RETURNS void
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
-- Name: merge_sandboxes(character varying, timestamp with time zone, json, boolean); Type: FUNCTION; Schema: goiardi; Owner: -
--

CREATE FUNCTION merge_sandboxes(m_sbox_id character varying, m_creation_time timestamp with time zone, m_checksums json, m_completed boolean) RETURNS void
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
-- Name: clients; Type: TABLE; Schema: goiardi; Owner: -; Tablespace: 
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
-- Name: cookbook_versions; Type: TABLE; Schema: goiardi; Owner: -; Tablespace: 
--

CREATE TABLE cookbook_versions (
    id bigint NOT NULL,
    cookbook_id bigint NOT NULL,
    major_ver bigint NOT NULL,
    minor_ver bigint NOT NULL,
    patch_ver bigint DEFAULT 0 NOT NULL,
    frozen boolean,
    metadata json,
    definitions json,
    libraries json,
    attributes json,
    recipes json,
    providers json,
    resources json,
    templates json,
    root_files json,
    files json,
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
-- Name: cookbooks; Type: TABLE; Schema: goiardi; Owner: -; Tablespace: 
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
-- Name: data_bag_items; Type: TABLE; Schema: goiardi; Owner: -; Tablespace: 
--

CREATE TABLE data_bag_items (
    id bigint NOT NULL,
    name text NOT NULL,
    orig_name text NOT NULL,
    data_bag_id bigint NOT NULL,
    raw_data json,
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
-- Name: data_bags; Type: TABLE; Schema: goiardi; Owner: -; Tablespace: 
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
-- Name: environments; Type: TABLE; Schema: goiardi; Owner: -; Tablespace: 
--

CREATE TABLE environments (
    id bigint NOT NULL,
    name text,
    organization_id bigint DEFAULT 1 NOT NULL,
    description text,
    default_attr json,
    override_attr json,
    cookbook_vers json,
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
-- Name: file_checksums; Type: TABLE; Schema: goiardi; Owner: -; Tablespace: 
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
-- Name: log_infos; Type: TABLE; Schema: goiardi; Owner: -; Tablespace: 
--

CREATE TABLE log_infos (
    id bigint NOT NULL,
    actor_id bigint DEFAULT 0 NOT NULL,
    actor_info text,
    actor_type log_actor NOT NULL,
    organization_id bigint DEFAULT 1::bigint NOT NULL,
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
-- Name: nodes; Type: TABLE; Schema: goiardi; Owner: -; Tablespace: 
--

CREATE TABLE nodes (
    id bigint NOT NULL,
    name text NOT NULL,
    organization_id bigint DEFAULT 1 NOT NULL,
    chef_environment text DEFAULT '_default'::text NOT NULL,
    run_list json,
    automatic_attr json,
    normal_attr json,
    default_attr json,
    override_attr json,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


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
-- Name: organizations; Type: TABLE; Schema: goiardi; Owner: -; Tablespace: 
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
-- Name: reports; Type: TABLE; Schema: goiardi; Owner: -; Tablespace: 
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
    resources json,
    data json,
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
-- Name: roles; Type: TABLE; Schema: goiardi; Owner: -; Tablespace: 
--

CREATE TABLE roles (
    id bigint NOT NULL,
    name text NOT NULL,
    organization_id bigint DEFAULT 1 NOT NULL,
    description text,
    run_list json,
    env_run_lists json,
    default_attr json,
    override_attr json,
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
-- Name: sandboxes; Type: TABLE; Schema: goiardi; Owner: -; Tablespace: 
--

CREATE TABLE sandboxes (
    id bigint NOT NULL,
    sbox_id character varying(32) NOT NULL,
    organization_id bigint DEFAULT 1 NOT NULL,
    creation_time timestamp with time zone NOT NULL,
    checksums json,
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
-- Name: users; Type: TABLE; Schema: goiardi; Owner: -; Tablespace: 
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
-- Name: changes; Type: TABLE; Schema: sqitch; Owner: -; Tablespace: 
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
    planner_email text NOT NULL
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
-- Name: dependencies; Type: TABLE; Schema: sqitch; Owner: -; Tablespace: 
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
-- Name: events; Type: TABLE; Schema: sqitch; Owner: -; Tablespace: 
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
    CONSTRAINT events_event_check CHECK ((event = ANY (ARRAY['deploy'::text, 'revert'::text, 'fail'::text])))
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
-- Name: projects; Type: TABLE; Schema: sqitch; Owner: -; Tablespace: 
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
-- Name: tags; Type: TABLE; Schema: sqitch; Owner: -; Tablespace: 
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
1	_default	1	The default Chef environment	\N	\N	\N	2014-07-20 23:17:09.177003-07	2014-07-20 23:17:09.177003-07
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
-- Data for Name: nodes; Type: TABLE DATA; Schema: goiardi; Owner: -
--

COPY nodes (id, name, organization_id, chef_environment, run_list, automatic_attr, normal_attr, default_attr, override_attr, created_at, updated_at) FROM stdin;
\.


--
-- Name: nodes_id_seq; Type: SEQUENCE SET; Schema: goiardi; Owner: -
--

SELECT pg_catalog.setval('nodes_id_seq', 1, false);


--
-- Data for Name: organizations; Type: TABLE DATA; Schema: goiardi; Owner: -
--

COPY organizations (id, name, description, created_at, updated_at) FROM stdin;
1	default	\N	2014-07-20 23:17:09.407193-07	2014-07-20 23:17:09.407193-07
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

COPY changes (change_id, change, project, note, committed_at, committer_name, committer_email, planned_at, planner_name, planner_email) FROM stdin;
c89b0e25c808b327036c88e6c9750c7526314c86	goiardi_schema	goiardi_postgres	Add schema for goiardi-postgres	2014-07-20 23:17:09.164895-07	Jeremy Bingham	jbingham@gmail.com	2014-05-27 14:09:07-07	Jeremy Bingham	jbingham@gmail.com
367c28670efddf25455b9fd33c23a5a278b08bb4	environments	goiardi_postgres	Environments for postgres	2014-07-20 23:17:09.184779-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 00:40:11-07	Jeremy Bingham	jbingham@gmail.com
911c456769628c817340ee77fc8d2b7c1d697782	nodes	goiardi_postgres	Create node table	2014-07-20 23:17:09.20439-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 10:37:46-07	Jeremy Bingham	jbingham@gmail.com
faa3571aa479de60f25785e707433b304ba3d2c7	clients	goiardi_postgres	Create client table	2014-07-20 23:17:09.222716-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 11:05:33-07	Jeremy Bingham	jbingham@gmail.com
bb82d8869ffca8ba3d03a1502c50dbb3eee7a2e0	users	goiardi_postgres	Create user table	2014-07-20 23:17:09.248893-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 11:15:02-07	Jeremy Bingham	jbingham@gmail.com
138bc49d92c0bbb024cea41532a656f2d7f9b072	cookbooks	goiardi_postgres	Create cookbook  table	2014-07-20 23:17:09.267547-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 11:27:27-07	Jeremy Bingham	jbingham@gmail.com
f529038064a0259bdecbdab1f9f665e17ddb6136	cookbook_versions	goiardi_postgres	Create cookbook versions table	2014-07-20 23:17:09.28688-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 11:31:34-07	Jeremy Bingham	jbingham@gmail.com
85483913f96710c1267c6abacb6568cef9327f15	data_bags	goiardi_postgres	Create cookbook data bags table	2014-07-20 23:17:09.303974-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 11:42:04-07	Jeremy Bingham	jbingham@gmail.com
feddf91b62caed36c790988bd29222591980433b	data_bag_items	goiardi_postgres	Create data bag items table	2014-07-20 23:17:09.323406-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 12:02:31-07	Jeremy Bingham	jbingham@gmail.com
6a4489d9436ba1541d272700b303410cc906b08f	roles	goiardi_postgres	Create roles table	2014-07-20 23:17:09.342653-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 12:09:28-07	Jeremy Bingham	jbingham@gmail.com
c4b32778f2911930f583ce15267aade320ac4dcd	sandboxes	goiardi_postgres	Create sandboxes table	2014-07-20 23:17:09.36161-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 12:14:48-07	Jeremy Bingham	jbingham@gmail.com
81003655b93b41359804027fc202788aa0ddd9a9	log_infos	goiardi_postgres	Create log_infos table	2014-07-20 23:17:09.394103-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 12:19:10-07	Jeremy Bingham	jbingham@gmail.com
fce5b7aeed2ad742de1309d7841577cff19475a7	organizations	goiardi_postgres	Create organizations table	2014-07-20 23:17:09.414544-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 12:46:28-07	Jeremy Bingham	jbingham@gmail.com
f2621482d1c130ea8fee15d09f966685409bf67c	file_checksums	goiardi_postgres	Create file checksums table	2014-07-20 23:17:09.431561-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 12:49:19-07	Jeremy Bingham	jbingham@gmail.com
db1eb360cd5e6449a468ceb781d82b45dafb5c2d	reports	goiardi_postgres	Create reports table	2014-07-20 23:17:09.453019-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 13:02:49-07	Jeremy Bingham	jbingham@gmail.com
c8b38382f7e5a18f36c621327f59205aa8aa9849	client_insert_duplicate	goiardi_postgres	Function to emulate insert ... on duplicate update for clients	2014-07-20 23:17:09.469343-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 23:00:04-07	Jeremy Bingham	jbingham@gmail.com
30774a960a0efb6adfbb1d526b8cdb1a45c7d039	client_rename	goiardi_postgres	Function to rename clients	2014-07-20 23:17:09.484817-07	Jeremy Bingham	jbingham@gmail.com	2014-05-30 10:22:50-07	Jeremy Bingham	jbingham@gmail.com
2d1fdc8128b0632e798df7346e76f122ed5915ec	user_insert_duplicate	goiardi_postgres	Function to emulate insert ... on duplicate update for clients	2014-07-20 23:17:09.499642-07	Jeremy Bingham	jbingham@gmail.com	2014-05-30 13:07:46-07	Jeremy Bingham	jbingham@gmail.com
f336c149ab32530c9c6ae4408c11558a635f39a1	user_rename	goiardi_postgres	Function to rename users	2014-07-20 23:17:09.51433-07	Jeremy Bingham	jbingham@gmail.com	2014-05-30 13:15:45-07	Jeremy Bingham	jbingham@gmail.com
841a7d554d44f9d0d0b8a1a5a9d0a06ce71a2453	cookbook_insert_update	goiardi_postgres	Cookbook insert/update	2014-07-20 23:17:09.532595-07	Jeremy Bingham	jbingham@gmail.com	2014-05-30 23:55:23-07	Jeremy Bingham	jbingham@gmail.com
085e2f6281914c9fa6521d59fea81f16c106b59f	cookbook_versions_insert_update	goiardi_postgres	Cookbook versions insert/update	2014-07-20 23:17:09.553079-07	Jeremy Bingham	jbingham@gmail.com	2014-05-30 23:56:05-07	Jeremy Bingham	jbingham@gmail.com
04bea39d649e4187d9579bd946fd60f760240d10	data_bag_insert_update	goiardi_postgres	Insert/update data bags	2014-07-20 23:17:09.567865-07	Jeremy Bingham	jbingham@gmail.com	2014-05-31 23:25:44-07	Jeremy Bingham	jbingham@gmail.com
092885e8b5d94a9c1834bf309e02dc0f955ff053	environment_insert_update	goiardi_postgres	Insert/update environments	2014-07-20 23:17:09.584468-07	Jeremy Bingham	jbingham@gmail.com	2014-06-01 12:55:34-07	Jeremy Bingham	jbingham@gmail.com
6d9587fa4275827c93ca9d7e0166ad1887b76cad	file_checksum_insert_ignore	goiardi_postgres	Insert ignore for file checksums	2014-07-20 23:17:09.600188-07	Jeremy Bingham	jbingham@gmail.com	2014-06-01 23:13:48-07	Jeremy Bingham	jbingham@gmail.com
82a95e5e6cbd8ba51fea33506e1edb2a12e37a92	node_insert_update	goiardi_postgres	Insert/update for nodes	2014-07-20 23:17:09.615796-07	Jeremy Bingham	jbingham@gmail.com	2014-06-01 23:25:20-07	Jeremy Bingham	jbingham@gmail.com
d052a8267a6512581e5cab1f89a2456f279727b9	report_insert_update	goiardi_postgres	Insert/update for reports	2014-07-20 23:17:09.629915-07	Jeremy Bingham	jbingham@gmail.com	2014-06-02 10:10:25-07	Jeremy Bingham	jbingham@gmail.com
acf76029633d50febbec7c4763b7173078eddaf7	role_insert_update	goiardi_postgres	Insert/update for roles	2014-07-20 23:17:09.6459-07	Jeremy Bingham	jbingham@gmail.com	2014-06-02 10:27:32-07	Jeremy Bingham	jbingham@gmail.com
b8ef36df686397ecb0fe67eb097e84aa0d78ac6b	sandbox_insert_update	goiardi_postgres	Insert/update for sandboxes	2014-07-20 23:17:09.661752-07	Jeremy Bingham	jbingham@gmail.com	2014-06-02 10:34:39-07	Jeremy Bingham	jbingham@gmail.com
93dbbda50a25da0a586e89ccee8fcfa2ddcb7c64	data_bag_item_insert	goiardi_postgres	Insert for data bag items	2014-07-20 23:17:09.676012-07	Jeremy Bingham	jbingham@gmail.com	2014-06-02 14:03:22-07	Jeremy Bingham	jbingham@gmail.com
c80c561c22f6e139165cdb338c7ce6fff8ff268d	bytea_to_json	goiardi_postgres	Change most postgres bytea fields to json, because in this peculiar case json is way faster than gob	2014-07-20 23:17:09.723813-07	Jeremy Bingham	jbingham@gmail.com	2014-07-20 02:41:22-07	Jeremy Bingham	jbingham@gmail.com
9966894e0fc0da573243f6a3c0fc1432a2b63043	joined_cookbkook_version	goiardi_postgres	a convenient view for joined versions for cookbook versions, adapted from erchef's joined_cookbook_version	2014-07-20 23:17:09.746519-07	Jeremy Bingham	jbingham@gmail.com	2014-07-20 03:21:28-07	Jeremy Bingham	jbingham@gmail.com
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
\.


--
-- Data for Name: events; Type: TABLE DATA; Schema: sqitch; Owner: -
--

COPY events (event, change_id, change, project, note, requires, conflicts, tags, committed_at, committer_name, committer_email, planned_at, planner_name, planner_email) FROM stdin;
deploy	c89b0e25c808b327036c88e6c9750c7526314c86	goiardi_schema	goiardi_postgres	Add schema for goiardi-postgres	{}	{}	{}	2014-07-20 23:17:09.166682-07	Jeremy Bingham	jbingham@gmail.com	2014-05-27 14:09:07-07	Jeremy Bingham	jbingham@gmail.com
deploy	367c28670efddf25455b9fd33c23a5a278b08bb4	environments	goiardi_postgres	Environments for postgres	{goiardi_schema}	{}	{}	2014-07-20 23:17:09.186612-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 00:40:11-07	Jeremy Bingham	jbingham@gmail.com
deploy	911c456769628c817340ee77fc8d2b7c1d697782	nodes	goiardi_postgres	Create node table	{goiardi_schema}	{}	{}	2014-07-20 23:17:09.206-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 10:37:46-07	Jeremy Bingham	jbingham@gmail.com
deploy	faa3571aa479de60f25785e707433b304ba3d2c7	clients	goiardi_postgres	Create client table	{goiardi_schema}	{}	{}	2014-07-20 23:17:09.224224-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 11:05:33-07	Jeremy Bingham	jbingham@gmail.com
deploy	bb82d8869ffca8ba3d03a1502c50dbb3eee7a2e0	users	goiardi_postgres	Create user table	{goiardi_schema}	{}	{}	2014-07-20 23:17:09.251003-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 11:15:02-07	Jeremy Bingham	jbingham@gmail.com
deploy	138bc49d92c0bbb024cea41532a656f2d7f9b072	cookbooks	goiardi_postgres	Create cookbook  table	{goiardi_schema}	{}	{}	2014-07-20 23:17:09.268882-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 11:27:27-07	Jeremy Bingham	jbingham@gmail.com
deploy	f529038064a0259bdecbdab1f9f665e17ddb6136	cookbook_versions	goiardi_postgres	Create cookbook versions table	{cookbooks,goiardi_schema}	{}	{}	2014-07-20 23:17:09.288446-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 11:31:34-07	Jeremy Bingham	jbingham@gmail.com
deploy	85483913f96710c1267c6abacb6568cef9327f15	data_bags	goiardi_postgres	Create cookbook data bags table	{goiardi_schema}	{}	{}	2014-07-20 23:17:09.305284-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 11:42:04-07	Jeremy Bingham	jbingham@gmail.com
deploy	feddf91b62caed36c790988bd29222591980433b	data_bag_items	goiardi_postgres	Create data bag items table	{data_bags,goiardi_schema}	{}	{}	2014-07-20 23:17:09.324692-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 12:02:31-07	Jeremy Bingham	jbingham@gmail.com
deploy	6a4489d9436ba1541d272700b303410cc906b08f	roles	goiardi_postgres	Create roles table	{goiardi_schema}	{}	{}	2014-07-20 23:17:09.34392-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 12:09:28-07	Jeremy Bingham	jbingham@gmail.com
deploy	c4b32778f2911930f583ce15267aade320ac4dcd	sandboxes	goiardi_postgres	Create sandboxes table	{goiardi_schema}	{}	{}	2014-07-20 23:17:09.362781-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 12:14:48-07	Jeremy Bingham	jbingham@gmail.com
deploy	81003655b93b41359804027fc202788aa0ddd9a9	log_infos	goiardi_postgres	Create log_infos table	{clients,users,goiardi_schema}	{}	{}	2014-07-20 23:17:09.395597-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 12:19:10-07	Jeremy Bingham	jbingham@gmail.com
deploy	fce5b7aeed2ad742de1309d7841577cff19475a7	organizations	goiardi_postgres	Create organizations table	{}	{}	{}	2014-07-20 23:17:09.415599-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 12:46:28-07	Jeremy Bingham	jbingham@gmail.com
deploy	f2621482d1c130ea8fee15d09f966685409bf67c	file_checksums	goiardi_postgres	Create file checksums table	{}	{}	{}	2014-07-20 23:17:09.432551-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 12:49:19-07	Jeremy Bingham	jbingham@gmail.com
deploy	db1eb360cd5e6449a468ceb781d82b45dafb5c2d	reports	goiardi_postgres	Create reports table	{}	{}	{}	2014-07-20 23:17:09.454103-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 13:02:49-07	Jeremy Bingham	jbingham@gmail.com
deploy	c8b38382f7e5a18f36c621327f59205aa8aa9849	client_insert_duplicate	goiardi_postgres	Function to emulate insert ... on duplicate update for clients	{clients,goiardi_schema}	{}	{}	2014-07-20 23:17:09.470733-07	Jeremy Bingham	jbingham@gmail.com	2014-05-29 23:00:04-07	Jeremy Bingham	jbingham@gmail.com
deploy	30774a960a0efb6adfbb1d526b8cdb1a45c7d039	client_rename	goiardi_postgres	Function to rename clients	{clients,goiardi_schema}	{}	{}	2014-07-20 23:17:09.486128-07	Jeremy Bingham	jbingham@gmail.com	2014-05-30 10:22:50-07	Jeremy Bingham	jbingham@gmail.com
deploy	2d1fdc8128b0632e798df7346e76f122ed5915ec	user_insert_duplicate	goiardi_postgres	Function to emulate insert ... on duplicate update for clients	{users,goiardi_schema}	{}	{}	2014-07-20 23:17:09.500835-07	Jeremy Bingham	jbingham@gmail.com	2014-05-30 13:07:46-07	Jeremy Bingham	jbingham@gmail.com
deploy	f336c149ab32530c9c6ae4408c11558a635f39a1	user_rename	goiardi_postgres	Function to rename users	{users,goiardi_schema}	{}	{}	2014-07-20 23:17:09.515652-07	Jeremy Bingham	jbingham@gmail.com	2014-05-30 13:15:45-07	Jeremy Bingham	jbingham@gmail.com
deploy	841a7d554d44f9d0d0b8a1a5a9d0a06ce71a2453	cookbook_insert_update	goiardi_postgres	Cookbook insert/update	{cookbooks,goiardi_schema}	{}	{}	2014-07-20 23:17:09.53494-07	Jeremy Bingham	jbingham@gmail.com	2014-05-30 23:55:23-07	Jeremy Bingham	jbingham@gmail.com
deploy	085e2f6281914c9fa6521d59fea81f16c106b59f	cookbook_versions_insert_update	goiardi_postgres	Cookbook versions insert/update	{cookbook_versions,goiardi_schema}	{}	{}	2014-07-20 23:17:09.554344-07	Jeremy Bingham	jbingham@gmail.com	2014-05-30 23:56:05-07	Jeremy Bingham	jbingham@gmail.com
deploy	04bea39d649e4187d9579bd946fd60f760240d10	data_bag_insert_update	goiardi_postgres	Insert/update data bags	{data_bags,goiardi_schema}	{}	{}	2014-07-20 23:17:09.569851-07	Jeremy Bingham	jbingham@gmail.com	2014-05-31 23:25:44-07	Jeremy Bingham	jbingham@gmail.com
deploy	092885e8b5d94a9c1834bf309e02dc0f955ff053	environment_insert_update	goiardi_postgres	Insert/update environments	{environments,goiardi_schema}	{}	{}	2014-07-20 23:17:09.585744-07	Jeremy Bingham	jbingham@gmail.com	2014-06-01 12:55:34-07	Jeremy Bingham	jbingham@gmail.com
deploy	6d9587fa4275827c93ca9d7e0166ad1887b76cad	file_checksum_insert_ignore	goiardi_postgres	Insert ignore for file checksums	{file_checksums,goiardi_schema}	{}	{}	2014-07-20 23:17:09.601702-07	Jeremy Bingham	jbingham@gmail.com	2014-06-01 23:13:48-07	Jeremy Bingham	jbingham@gmail.com
deploy	82a95e5e6cbd8ba51fea33506e1edb2a12e37a92	node_insert_update	goiardi_postgres	Insert/update for nodes	{nodes,goiardi_schema}	{}	{}	2014-07-20 23:17:09.617035-07	Jeremy Bingham	jbingham@gmail.com	2014-06-01 23:25:20-07	Jeremy Bingham	jbingham@gmail.com
deploy	d052a8267a6512581e5cab1f89a2456f279727b9	report_insert_update	goiardi_postgres	Insert/update for reports	{reports,goiardi_schema}	{}	{}	2014-07-20 23:17:09.631416-07	Jeremy Bingham	jbingham@gmail.com	2014-06-02 10:10:25-07	Jeremy Bingham	jbingham@gmail.com
deploy	acf76029633d50febbec7c4763b7173078eddaf7	role_insert_update	goiardi_postgres	Insert/update for roles	{roles,goiardi_schema}	{}	{}	2014-07-20 23:17:09.647316-07	Jeremy Bingham	jbingham@gmail.com	2014-06-02 10:27:32-07	Jeremy Bingham	jbingham@gmail.com
deploy	b8ef36df686397ecb0fe67eb097e84aa0d78ac6b	sandbox_insert_update	goiardi_postgres	Insert/update for sandboxes	{sandboxes,goiardi_schema}	{}	{}	2014-07-20 23:17:09.663184-07	Jeremy Bingham	jbingham@gmail.com	2014-06-02 10:34:39-07	Jeremy Bingham	jbingham@gmail.com
deploy	93dbbda50a25da0a586e89ccee8fcfa2ddcb7c64	data_bag_item_insert	goiardi_postgres	Insert for data bag items	{data_bag_items,data_bags,goiardi_schema}	{}	{@v0.6.0}	2014-07-20 23:17:09.67916-07	Jeremy Bingham	jbingham@gmail.com	2014-06-02 14:03:22-07	Jeremy Bingham	jbingham@gmail.com
deploy	c80c561c22f6e139165cdb338c7ce6fff8ff268d	bytea_to_json	goiardi_postgres	Change most postgres bytea fields to json, because in this peculiar case json is way faster than gob	{}	{}	{}	2014-07-20 23:17:09.725431-07	Jeremy Bingham	jbingham@gmail.com	2014-07-20 02:41:22-07	Jeremy Bingham	jbingham@gmail.com
deploy	9966894e0fc0da573243f6a3c0fc1432a2b63043	joined_cookbkook_version	goiardi_postgres	a convenient view for joined versions for cookbook versions, adapted from erchef's joined_cookbook_version	{}	{}	{}	2014-07-20 23:17:09.747915-07	Jeremy Bingham	jbingham@gmail.com	2014-07-20 03:21:28-07	Jeremy Bingham	jbingham@gmail.com
\.


--
-- Data for Name: projects; Type: TABLE DATA; Schema: sqitch; Owner: -
--

COPY projects (project, uri, created_at, creator_name, creator_email) FROM stdin;
goiardi_postgres	http://ctdk.github.com/goiardi/postgres-support	2014-07-20 23:17:09.140404-07	Jeremy Bingham	jbingham@gmail.com
\.


--
-- Data for Name: tags; Type: TABLE DATA; Schema: sqitch; Owner: -
--

COPY tags (tag_id, tag, project, change_id, note, committed_at, committer_name, committer_email, planned_at, planner_name, planner_email) FROM stdin;
fd6ca4c1426a85718d19687591885a2c2a516952	@v0.6.0	goiardi_postgres	93dbbda50a25da0a586e89ccee8fcfa2ddcb7c64	Tag v0.6.0 for release	2014-07-20 23:17:09.677436-07	Jeremy Bingham	jbingham@gmail.com	2014-06-27 00:20:56-07	Jeremy Bingham	jbingham@gmail.com
\.


SET search_path = goiardi, pg_catalog;

--
-- Name: clients_organization_id_name_key; Type: CONSTRAINT; Schema: goiardi; Owner: -; Tablespace: 
--

ALTER TABLE ONLY clients
    ADD CONSTRAINT clients_organization_id_name_key UNIQUE (organization_id, name);


--
-- Name: clients_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -; Tablespace: 
--

ALTER TABLE ONLY clients
    ADD CONSTRAINT clients_pkey PRIMARY KEY (id);


--
-- Name: cookbook_versions_cookbook_id_major_ver_minor_ver_patch_ver_key; Type: CONSTRAINT; Schema: goiardi; Owner: -; Tablespace: 
--

ALTER TABLE ONLY cookbook_versions
    ADD CONSTRAINT cookbook_versions_cookbook_id_major_ver_minor_ver_patch_ver_key UNIQUE (cookbook_id, major_ver, minor_ver, patch_ver);


--
-- Name: cookbook_versions_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -; Tablespace: 
--

ALTER TABLE ONLY cookbook_versions
    ADD CONSTRAINT cookbook_versions_pkey PRIMARY KEY (id);


--
-- Name: cookbooks_organization_id_name_key; Type: CONSTRAINT; Schema: goiardi; Owner: -; Tablespace: 
--

ALTER TABLE ONLY cookbooks
    ADD CONSTRAINT cookbooks_organization_id_name_key UNIQUE (organization_id, name);


--
-- Name: cookbooks_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -; Tablespace: 
--

ALTER TABLE ONLY cookbooks
    ADD CONSTRAINT cookbooks_pkey PRIMARY KEY (id);


--
-- Name: data_bag_items_data_bag_id_name_key; Type: CONSTRAINT; Schema: goiardi; Owner: -; Tablespace: 
--

ALTER TABLE ONLY data_bag_items
    ADD CONSTRAINT data_bag_items_data_bag_id_name_key UNIQUE (data_bag_id, name);


--
-- Name: data_bag_items_data_bag_id_orig_name_key; Type: CONSTRAINT; Schema: goiardi; Owner: -; Tablespace: 
--

ALTER TABLE ONLY data_bag_items
    ADD CONSTRAINT data_bag_items_data_bag_id_orig_name_key UNIQUE (data_bag_id, orig_name);


--
-- Name: data_bag_items_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -; Tablespace: 
--

ALTER TABLE ONLY data_bag_items
    ADD CONSTRAINT data_bag_items_pkey PRIMARY KEY (id);


--
-- Name: data_bags_organization_id_name_key; Type: CONSTRAINT; Schema: goiardi; Owner: -; Tablespace: 
--

ALTER TABLE ONLY data_bags
    ADD CONSTRAINT data_bags_organization_id_name_key UNIQUE (organization_id, name);


--
-- Name: data_bags_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -; Tablespace: 
--

ALTER TABLE ONLY data_bags
    ADD CONSTRAINT data_bags_pkey PRIMARY KEY (id);


--
-- Name: environments_organization_id_name_key; Type: CONSTRAINT; Schema: goiardi; Owner: -; Tablespace: 
--

ALTER TABLE ONLY environments
    ADD CONSTRAINT environments_organization_id_name_key UNIQUE (organization_id, name);


--
-- Name: environments_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -; Tablespace: 
--

ALTER TABLE ONLY environments
    ADD CONSTRAINT environments_pkey PRIMARY KEY (id);


--
-- Name: file_checksums_organization_id_checksum_key; Type: CONSTRAINT; Schema: goiardi; Owner: -; Tablespace: 
--

ALTER TABLE ONLY file_checksums
    ADD CONSTRAINT file_checksums_organization_id_checksum_key UNIQUE (organization_id, checksum);


--
-- Name: file_checksums_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -; Tablespace: 
--

ALTER TABLE ONLY file_checksums
    ADD CONSTRAINT file_checksums_pkey PRIMARY KEY (id);


--
-- Name: log_infos_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -; Tablespace: 
--

ALTER TABLE ONLY log_infos
    ADD CONSTRAINT log_infos_pkey PRIMARY KEY (id);


--
-- Name: nodes_organization_id_name_key; Type: CONSTRAINT; Schema: goiardi; Owner: -; Tablespace: 
--

ALTER TABLE ONLY nodes
    ADD CONSTRAINT nodes_organization_id_name_key UNIQUE (organization_id, name);


--
-- Name: nodes_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -; Tablespace: 
--

ALTER TABLE ONLY nodes
    ADD CONSTRAINT nodes_pkey PRIMARY KEY (id);


--
-- Name: organizations_name_key; Type: CONSTRAINT; Schema: goiardi; Owner: -; Tablespace: 
--

ALTER TABLE ONLY organizations
    ADD CONSTRAINT organizations_name_key UNIQUE (name);


--
-- Name: organizations_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -; Tablespace: 
--

ALTER TABLE ONLY organizations
    ADD CONSTRAINT organizations_pkey PRIMARY KEY (id);


--
-- Name: reports_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -; Tablespace: 
--

ALTER TABLE ONLY reports
    ADD CONSTRAINT reports_pkey PRIMARY KEY (id);


--
-- Name: reports_run_id_key; Type: CONSTRAINT; Schema: goiardi; Owner: -; Tablespace: 
--

ALTER TABLE ONLY reports
    ADD CONSTRAINT reports_run_id_key UNIQUE (run_id);


--
-- Name: roles_organization_id_name_key; Type: CONSTRAINT; Schema: goiardi; Owner: -; Tablespace: 
--

ALTER TABLE ONLY roles
    ADD CONSTRAINT roles_organization_id_name_key UNIQUE (organization_id, name);


--
-- Name: roles_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -; Tablespace: 
--

ALTER TABLE ONLY roles
    ADD CONSTRAINT roles_pkey PRIMARY KEY (id);


--
-- Name: sandboxes_organization_id_sbox_id_key; Type: CONSTRAINT; Schema: goiardi; Owner: -; Tablespace: 
--

ALTER TABLE ONLY sandboxes
    ADD CONSTRAINT sandboxes_organization_id_sbox_id_key UNIQUE (organization_id, sbox_id);


--
-- Name: sandboxes_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -; Tablespace: 
--

ALTER TABLE ONLY sandboxes
    ADD CONSTRAINT sandboxes_pkey PRIMARY KEY (id);


--
-- Name: users_email_key; Type: CONSTRAINT; Schema: goiardi; Owner: -; Tablespace: 
--

ALTER TABLE ONLY users
    ADD CONSTRAINT users_email_key UNIQUE (email);


--
-- Name: users_name_key; Type: CONSTRAINT; Schema: goiardi; Owner: -; Tablespace: 
--

ALTER TABLE ONLY users
    ADD CONSTRAINT users_name_key UNIQUE (name);


--
-- Name: users_pkey; Type: CONSTRAINT; Schema: goiardi; Owner: -; Tablespace: 
--

ALTER TABLE ONLY users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


SET search_path = sqitch, pg_catalog;

--
-- Name: changes_pkey; Type: CONSTRAINT; Schema: sqitch; Owner: -; Tablespace: 
--

ALTER TABLE ONLY changes
    ADD CONSTRAINT changes_pkey PRIMARY KEY (change_id);


--
-- Name: dependencies_pkey; Type: CONSTRAINT; Schema: sqitch; Owner: -; Tablespace: 
--

ALTER TABLE ONLY dependencies
    ADD CONSTRAINT dependencies_pkey PRIMARY KEY (change_id, dependency);


--
-- Name: events_pkey; Type: CONSTRAINT; Schema: sqitch; Owner: -; Tablespace: 
--

ALTER TABLE ONLY events
    ADD CONSTRAINT events_pkey PRIMARY KEY (change_id, committed_at);


--
-- Name: projects_pkey; Type: CONSTRAINT; Schema: sqitch; Owner: -; Tablespace: 
--

ALTER TABLE ONLY projects
    ADD CONSTRAINT projects_pkey PRIMARY KEY (project);


--
-- Name: projects_uri_key; Type: CONSTRAINT; Schema: sqitch; Owner: -; Tablespace: 
--

ALTER TABLE ONLY projects
    ADD CONSTRAINT projects_uri_key UNIQUE (uri);


--
-- Name: tags_pkey; Type: CONSTRAINT; Schema: sqitch; Owner: -; Tablespace: 
--

ALTER TABLE ONLY tags
    ADD CONSTRAINT tags_pkey PRIMARY KEY (tag_id);


--
-- Name: tags_project_tag_key; Type: CONSTRAINT; Schema: sqitch; Owner: -; Tablespace: 
--

ALTER TABLE ONLY tags
    ADD CONSTRAINT tags_project_tag_key UNIQUE (project, tag);


SET search_path = goiardi, pg_catalog;

--
-- Name: log_info_orgs; Type: INDEX; Schema: goiardi; Owner: -; Tablespace: 
--

CREATE INDEX log_info_orgs ON log_infos USING btree (organization_id);


--
-- Name: log_infos_action; Type: INDEX; Schema: goiardi; Owner: -; Tablespace: 
--

CREATE INDEX log_infos_action ON log_infos USING btree (action);


--
-- Name: log_infos_actor; Type: INDEX; Schema: goiardi; Owner: -; Tablespace: 
--

CREATE INDEX log_infos_actor ON log_infos USING btree (actor_id);


--
-- Name: log_infos_obj; Type: INDEX; Schema: goiardi; Owner: -; Tablespace: 
--

CREATE INDEX log_infos_obj ON log_infos USING btree (object_type, object_name);


--
-- Name: log_infos_time; Type: INDEX; Schema: goiardi; Owner: -; Tablespace: 
--

CREATE INDEX log_infos_time ON log_infos USING btree ("time");


--
-- Name: nodes_chef_env; Type: INDEX; Schema: goiardi; Owner: -; Tablespace: 
--

CREATE INDEX nodes_chef_env ON nodes USING btree (chef_environment);


--
-- Name: report_node_organization; Type: INDEX; Schema: goiardi; Owner: -; Tablespace: 
--

CREATE INDEX report_node_organization ON reports USING btree (node_name, organization_id);


--
-- Name: report_organization_id; Type: INDEX; Schema: goiardi; Owner: -; Tablespace: 
--

CREATE INDEX report_organization_id ON reports USING btree (organization_id);


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
-- Name: public; Type: ACL; Schema: -; Owner: -
--

REVOKE ALL ON SCHEMA public FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM jeremy;
GRANT ALL ON SCHEMA public TO jeremy;
GRANT ALL ON SCHEMA public TO PUBLIC;


--
-- PostgreSQL database dump complete
--

