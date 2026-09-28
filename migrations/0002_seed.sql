-- Demo data. hospital-a has an HIS API (served by the mock-his container in
-- docker compose); hospital-b has none, so it only has locally stored patients.
INSERT INTO hospitals (code, name, his_base_url) VALUES
    ('hospital-a', 'Hospital A', 'http://mock-his-a:8081'),
    ('hospital-b', 'Hospital B', NULL)
ON CONFLICT (code) DO NOTHING;

INSERT INTO patients (hospital_id, patient_hn, first_name_th, last_name_th, first_name_en, last_name_en,
                      date_of_birth, national_id, passport_id, phone_number, email, gender)
SELECT h.id, p.patient_hn, p.first_name_th, p.last_name_th, p.first_name_en, p.last_name_en,
       p.date_of_birth::DATE, p.national_id, p.passport_id, p.phone_number, p.email, p.gender
FROM hospitals h
JOIN (VALUES
    ('hospital-a', 'HN-A-0001', 'สมชาย', 'ใจดี',   'Somchai', 'Jaidee',   '1985-04-12', '1103700000011', NULL,        '0811111111', 'somchai@example.com', 'M'),
    ('hospital-a', 'HN-A-0002', 'สมหญิง', 'รักเรียน', 'Somying', 'Rakrian',  '1990-09-01', '1103700000022', NULL,        '0822222222', 'somying@example.com', 'F'),
    ('hospital-b', 'HN-B-0001', 'สมชาย', 'ใจดี',   'Somchai', 'Jaidee',   '1985-04-12', '1103700000011', NULL,        '0811111111', 'somchai@example.com', 'M'),
    ('hospital-b', 'HN-B-0002', NULL,     NULL,      'John',    'Smith',    '1978-01-30', NULL,            'AA1234567', '0833333333', 'john@example.com',    'M')
) AS p (hospital_code, patient_hn, first_name_th, last_name_th, first_name_en, last_name_en,
        date_of_birth, national_id, passport_id, phone_number, email, gender)
    ON h.code = p.hospital_code
ORDER BY h.id, p.patient_hn
ON CONFLICT (hospital_id, patient_hn) DO NOTHING;
