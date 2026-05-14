# `create_user`

This script creates a user in the database. This can be used to initialize the first user.

## How to run

`make create-user`

You can specify the following environment variables
- DB_URI: Connection string to connect to the database
- NAME: Name of the user to create 
- LOGIN_USERNAME: Username of the user to create
- LOGIN_PASSWORD: Password of the user to create. If left empty, a random password will be generated and printed
