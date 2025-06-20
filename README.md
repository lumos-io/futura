# Futura Monorepo

This monorepo contains all the code to make the OpisVigilant platform working with the except of the infrastructure

## Nix Development setup

This project uses `nix` to make it more portable and reproduciable.

### .env file

You need to create a `.env.local` file where you add secrets that shouldn't be part of the git commit. Nix will try to read the file and stop in case it cannot find it. A `.env.tmp` file is committed with the variables that need to be used.
