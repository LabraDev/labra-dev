{
	description = "The backend flake of Labra";

	inputs = { 
		nixpkgs.url = "github:nixos/nixpkgs";
		flake-utils.url = "github:numtide/flake-utils";
	};

	outputs = { self, nixpkgs, flake-utils }:
		flake-utils.lib.eachDefaultSystem (
		system:
			let pkgs = nixpkgs.legacyPackages.${system};
			in {
				devShell = pkgs.mkShell { 
					buildInputs = [ pkgs.postgresql pkgs.go pkgs.air ]; 

					shellHook = ''
						export PGDATA=$PWD/.postgres
						export PGHOST=$PGDATA
						export PGPORT=5433


						mkdir -p $PGDATA


						if [ ! -f "$PGDATA/PG_VERSION" ]; then
							echo " ---- initalizing postgres ---- "
							initdb -D $PGDATA --no-locale --encoding UTF8
						fi


						if ! pg_ctl -D $PGDATA status > /dev/null 2>&1; then
							echo " ---- starting postgres ---- "
							pg_ctl -D $PGDATA -o "-p $PGPORT -k $PGHOST" -l $PGDATA/log start
						fi

						if ! psql -h $PGHOST -p $PGPORT -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='labra'" | grep -q 1; then
							echo " ---- create labra database ----"
							createdb -h $PGHOST -p  $PGPORT labra
						fi

						if [ ! -f .env ]; then
							echo "dotenv not found, generating one from example"
							cp .env.example .env
						fi
						
						echo "Entered flake"
						echo "Postgres is running with host: $PGHOST | port: $PGPORT | on database: labra"

						trap "pg_ctl -D $PGDATA stop" EXIT
					'';

				};
			});
}
