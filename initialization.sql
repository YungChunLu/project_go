DROP TABLE IF EXISTS orders;
CREATE TABLE orders (id serial primary key, distance int not null, status text not null);
