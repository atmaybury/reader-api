#!/bin/bash

export DB_PATH=postgresql://andrew:WMI8fsHvYL0sR4hCOTGQ06zSxmoupIW9@dpg-cuo4sqrqf0us738rr4hg-a.singapore-postgres.render.com:5432/reader_db_z0oe && \
export JWT_SECRET=temporarySecret && \
go run .
