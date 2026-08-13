-- =====================================================
-- WALLET TABLES (PostgreSQL)
-- Style mengikuti tbl_admin / tbl_counter yang sudah ada
-- =====================================================

-- ================= TBL_USER =================
CREATE TABLE tbl_user (
	username varchar(30) NOT NULL,
	"password" varchar(250) NULL,
	"token" varchar(250) DEFAULT ''::character varying NULL,
	nama varchar(50) NULL,
	saldo numeric(18,2) DEFAULT 0 NOT NULL,
	status varchar(1) DEFAULT 'Y' NULL, -- Y=aktif, N=nonaktif
	create_by varchar(30) NULL,
	create_at timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
	update_by varchar(30) DEFAULT ''::character varying NULL,
	update_at timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
	CONSTRAINT tbl_user_pk PRIMARY KEY (username),
	CONSTRAINT tbl_user_saldo_ck CHECK (saldo >= 0)
);

-- ================= TBL_TRX_TRANSAKSI =================
-- tipe   : CREDIT = tambah saldo | DEBIT = kurangi saldo
-- source : DEPOSIT, WIN  -> CREDIT
--          WITHDRAW, BET -> DEBIT
CREATE TABLE tbl_trx_transaksi (
	idtrx varchar(50) DEFAULT to_char(CURRENT_TIMESTAMP, 'YYMMDD') || '-' || gen_random_uuid() NOT NULL, -- ex: 260729-a1b2c3d4-...
	notrx varchar(50) NOT NULL,               -- nomor trx (generate dari tbl_counter)
	username varchar(30) NOT NULL,
	tipe varchar(10) NOT NULL,                -- CREDIT / DEBIT
	"source" varchar(20) NOT NULL,            -- DEPOSIT / WITHDRAW / WIN / BET
	amount numeric(18,2) NOT NULL,
	saldo_before numeric(18,2) NOT NULL,
	saldo_after numeric(18,2) NOT NULL,
	refno varchar(50) DEFAULT ''::character varying NULL,  -- ref game/bank
	status varchar(1) DEFAULT '1' NULL,       -- 0=pending, 1=sukses, 2=gagal
	keterangan varchar(250) DEFAULT ''::character varying NULL,
	create_by varchar(30) NULL,
	create_at timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
	update_by varchar(30) DEFAULT ''::character varying NULL,
	update_at timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
	CONSTRAINT tbl_trx_transaksi_pk PRIMARY KEY (idtrx),
	CONSTRAINT tbl_trx_transaksi_un UNIQUE (notrx),
	CONSTRAINT tbl_trx_transaksi_user_fk FOREIGN KEY (username) REFERENCES tbl_user(username),
	CONSTRAINT tbl_trx_tipe_ck CHECK (tipe IN ('CREDIT','DEBIT')),
	CONSTRAINT tbl_trx_source_ck CHECK ("source" IN ('DEPOSIT','WITHDRAW','WIN','BET')),
	CONSTRAINT tbl_trx_amount_ck CHECK (amount > 0),
	-- pastikan kombinasi tipe & source valid
	CONSTRAINT tbl_trx_tipe_source_ck CHECK (
		(tipe = 'CREDIT' AND "source" IN ('DEPOSIT','WIN')) OR
		(tipe = 'DEBIT'  AND "source" IN ('WITHDRAW','BET'))
	)
);

CREATE INDEX idx_trx_username ON tbl_trx_transaksi (username);
CREATE INDEX idx_trx_create_at ON tbl_trx_transaksi (create_at);
CREATE INDEX idx_trx_tipe_source ON tbl_trx_transaksi (tipe, "source");

-- Generate notrx pakai sequence Postgres (nextval, atomic non-blocking) â€” BUKAN lagi
-- tbl_counter + "SELECT ... FOR UPDATE", soalnya row lock manual itu serialize SEMUA
-- transaksi wallet (deposit/withdraw/win/bet, semua user) lewat satu baris, jadi
-- bottleneck pas ada burst transaksi konkuren (timeout "error select counter").
CREATE SEQUENCE IF NOT EXISTS seq_notrx_transaksi;

-- =====================================================
-- CONTOH TRANSAKSI (jalankan dalam 1 database transaction)
-- =====================================================

-- DEPOSIT 100.000 (CREDIT)
-- BEGIN;
-- UPDATE tbl_user SET saldo = saldo + 100000, update_at = now() WHERE username = 'dev';
-- INSERT INTO tbl_trx_transaksi (notrx, username, tipe, "source", amount, saldo_before, saldo_after, create_by)
-- SELECT 'TRX-0001', 'dev', 'CREDIT', 'DEPOSIT', 100000, saldo - 100000, saldo, 'dev'
-- FROM tbl_user WHERE username = 'dev';
-- COMMIT;

-- BET 50.000 (DEBIT) — saldo tidak boleh minus (dijaga CHECK saldo >= 0)
-- BEGIN;
-- UPDATE tbl_user SET saldo = saldo - 50000, update_at = now() WHERE username = 'dev';
-- INSERT INTO tbl_trx_transaksi (notrx, username, tipe, "source", amount, saldo_before, saldo_after, refno, create_by)
-- SELECT 'TRX-0002', 'dev', 'DEBIT', 'BET', 50000, saldo + 50000, saldo, 'GAME-123', 'dev'
-- FROM tbl_user WHERE username = 'dev';
-- COMMIT;




CREATE TABLE tbl_admin (
	username varchar(30) NOT NULL,
	"password" varchar(250) NULL,
	idadmin varchar(30) NULL,
	"name" varchar(50) NULL,
	statuslogin varchar(1) NULL,
	lastlogin timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
	joindate date DEFAULT CURRENT_TIMESTAMP NOT NULL,
	ipaddress varchar(70) DEFAULT ''::character varying NULL,
	timezone varchar(30) DEFAULT ''::character varying NULL,
	create_by varchar(30) NULL,
	create_at timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
	update_by varchar(30) DEFAULT ''::character varying NULL,
	update_at timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
	CONSTRAINT tbl_admin_pk PRIMARY KEY (username)
);



CREATE TABLE public.tbl_counter (
	idcounter int4 DEFAULT nextval('idcounter_seq'::regclass) NOT NULL,
	nmcounter varchar(70) NULL,
	counter int8 NOT NULL,
	CONSTRAINT tbl_counter_pk PRIMARY KEY (idcounter)
);

-- ================= TBL_SLOW_QUERY =================
-- Query database yang lebih lambat dari threshold (lihat internal/connection/tracer.go)
-- otomatis kecatat di sini oleh pgx query tracer, ditampilin di halaman "Slow Query"
-- dashboard admin. Cuma nyimpen teks query (placeholder $1/$2/dst, BUKAN nilai asli
-- argumennya) supaya gak ada data sensitif (password/token) yang ke-log.
CREATE TABLE tbl_slow_query (
	id bigserial NOT NULL,
	query text NOT NULL,
	duration_ms int8 NOT NULL,
	create_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
	CONSTRAINT tbl_slow_query_pk PRIMARY KEY (id)
);

CREATE INDEX idx_slow_query_create_at ON tbl_slow_query (create_at DESC);