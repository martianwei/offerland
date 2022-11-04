### Helper

-   make help

### Database Setup

-   psql
-   CREATE DATABASE offerland;
-   CREATE ROLE offerland WITH LOGIN PASSWORD 'pa55word';
-   psql --host=localhost --dbname=offerland --username=offerland
-   make db/migrations/up
