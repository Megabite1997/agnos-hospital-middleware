-- Hospitals registered with the middleware. Each hospital may expose its own
-- HIS (Hospital Information System) API; his_base_url is NULL when it does not.
CREATE TABLE IF NOT EXISTS hospitals (
    id           BIGSERIAL PRIMARY KEY,
    code         VARCHAR(50)  NOT NULL UNIQUE,
    name         VARCHAR(255) NOT NULL,
    his_base_url VARCHAR(255),
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Hospital staff. A username is unique within a hospital, so two hospitals can
-- both have a staff member called "admin".
CREATE TABLE IF NOT EXISTS staff (
    id            BIGSERIAL PRIMARY KEY,
    hospital_id   BIGINT       NOT NULL REFERENCES hospitals (id) ON DELETE RESTRICT,
    username      VARCHAR(50)  NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_staff_hospital_username UNIQUE (hospital_id, username)
);

-- Patients mirror the HIS response shape. A patient row always belongs to one
-- hospital; the same person visiting two hospitals is two rows with two HNs.
CREATE TABLE IF NOT EXISTS patients (
    id             BIGSERIAL PRIMARY KEY,
    hospital_id    BIGINT       NOT NULL REFERENCES hospitals (id) ON DELETE RESTRICT,
    patient_hn     VARCHAR(50)  NOT NULL,
    first_name_th  VARCHAR(100),
    middle_name_th VARCHAR(100),
    last_name_th   VARCHAR(100),
    first_name_en  VARCHAR(100),
    middle_name_en VARCHAR(100),
    last_name_en   VARCHAR(100),
    date_of_birth  DATE,
    national_id    VARCHAR(13),
    passport_id    VARCHAR(20),
    phone_number   VARCHAR(20),
    email          VARCHAR(255),
    gender         CHAR(1) CHECK (gender IN ('M', 'F')),
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_patients_hospital_hn UNIQUE (hospital_id, patient_hn)
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_patients_hospital_national_id
    ON patients (hospital_id, national_id) WHERE national_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_patients_hospital_passport_id
    ON patients (hospital_id, passport_id) WHERE passport_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_patients_hospital_dob   ON patients (hospital_id, date_of_birth);
CREATE INDEX IF NOT EXISTS idx_patients_hospital_phone ON patients (hospital_id, phone_number);
CREATE INDEX IF NOT EXISTS idx_patients_hospital_email ON patients (hospital_id, LOWER(email));
