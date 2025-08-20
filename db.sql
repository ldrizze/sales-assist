--
-- PostgreSQL database dump
--

-- Dumped from database version 17.4 (Debian 17.4-1.pgdg120+2)
-- Dumped by pg_dump version 17.0

-- Started on 2025-05-01 19:01:12

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

DROP DATABASE salesassist;
--
-- TOC entry 3371 (class 1262 OID 24745)
-- Name: salesassist; Type: DATABASE; Schema: -; Owner: postgres
--

CREATE DATABASE salesassist WITH TEMPLATE = template0 ENCODING = 'UTF8' LOCALE_PROVIDER = libc LOCALE = 'en_US.utf8';


ALTER DATABASE salesassist OWNER TO postgres;

\connect salesassist

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- TOC entry 6 (class 2615 OID 24746)
-- Name: assist; Type: SCHEMA; Schema: -; Owner: postgres
--

CREATE SCHEMA assist;


ALTER SCHEMA assist OWNER TO postgres;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- TOC entry 219 (class 1259 OID 24754)
-- Name: chat_logs; Type: TABLE; Schema: assist; Owner: postgres
--

CREATE TABLE assist.chat_logs (
    id serial NOT NULL,
    phone_number character varying NOT NULL,
    data json,
    save_time timestamp without time zone DEFAULT now() NOT NULL
);


ALTER TABLE assist.chat_logs OWNER TO postgres;

--
-- TOC entry 218 (class 1259 OID 24747)
-- Name: suspended_chats; Type: TABLE; Schema: assist; Owner: postgres
--

CREATE TABLE assist.suspended_chats (
    phone_number character varying NOT NULL,
    data json
);


ALTER TABLE assist.suspended_chats OWNER TO postgres;

--
-- Name: orders; Type: TABLE; Schema: assist; Owner: postgres
--

CREATE TABLE assist.orders (
    id serial NOT NULL,
    phone_number character varying NOT NULL,
    data json
    created timestamp
);


ALTER TABLE assist.orders OWNER TO postgres;

--
-- TOC entry 3218 (class 2606 OID 24761)
-- Name: chat_logs chat_logs_pk; Type: CONSTRAINT; Schema: assist; Owner: postgres
--

ALTER TABLE ONLY assist.chat_logs
    ADD CONSTRAINT chat_logs_pk PRIMARY KEY (id, phone_number);


--
-- TOC entry 3216 (class 2606 OID 24753)
-- Name: suspended_chats suspended_chats_pk; Type: CONSTRAINT; Schema: assist; Owner: postgres
--

ALTER TABLE ONLY assist.suspended_chats
    ADD CONSTRAINT suspended_chats_pk PRIMARY KEY (phone_number);

--
-- Name: orders orders_pk; Type: CONSTRAINT; Schema: assist; Owner: postgres
--

ALTER TABLE ONLY assist.orders
    ADD CONSTRAINT orders_pk PRIMARY KEY (id, phone_number);


-- Completed on 2025-05-01 19:01:12

--
-- PostgreSQL database dump complete
--

