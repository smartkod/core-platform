-- 0007_taxonomy_seed: a minimal starter category tree.
--
-- Without at least one published node nobody can create an offering: the
-- catalogue requires a category reference and only a platform administrator may
-- add one. That would make a fresh installation look broken to the first
-- business that signs up.
--
-- These four roots are structural, not sector content: they describe the shape
-- of an offering (a service, a product, an event, a course) rather than any
-- particular industry. A platform administrator renames, extends or replaces
-- them from the admin endpoints; the ids are fixed so documentation and tests
-- can refer to them.

INSERT INTO taxonomy_nodes (id, parent_id, slug, name, description, status, published, sort, locale)
VALUES
    ('018f0000-0000-7000-8000-000000000001', NULL, 'hizmet',   'Hizmet',   'Yerinde veya uzaktan verilen hizmetler', 'active', true, 10, 'tr'),
    ('018f0000-0000-7000-8000-000000000002', NULL, 'urun',     'Ürün',     'Satışa sunulan fiziksel ürünler',        'active', true, 20, 'tr'),
    ('018f0000-0000-7000-8000-000000000003', NULL, 'etkinlik', 'Etkinlik', 'Tarih ve yeri olan etkinlikler',          'active', true, 30, 'tr'),
    ('018f0000-0000-7000-8000-000000000004', NULL, 'egitim',   'Eğitim',   'Kurs, ders ve eğitim programları',        'active', true, 40, 'tr')
ON CONFLICT (id) DO NOTHING;
