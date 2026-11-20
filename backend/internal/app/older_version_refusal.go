package app

// olderVersionRefusal follows the newer-schema error for the server, every console command and a
// startup recovery. It keeps the installation by default and describes the deliberate fresh
// install with an older image, whose cleanup Databasus never runs itself: the container cannot see
// how the host stores its data, so each example is labeled with the storage it fits.
const olderVersionRefusal = "Returning to an older Databasus version after an upgrade is not supported. " +
	"Start the image of the newer version instead to keep this installation.\n" +
	"\n" +
	"To use an older version anyway, start a fresh installation with it. This is not a downgrade: " +
	"it deletes every account, every setting, the master key and the backup files stored locally, " +
	"and the older version still cannot import Databasus backups made by the newer one. " +
	"Before you delete anything, copy every Databasus backup, the master key file and the local " +
	"backup files you still need to a place outside the storage you remove.\n" +
	"\n" +
	"Then remove the container and its data with the example that matches how the data is stored, " +
	"and recreate only the Databasus deployment with the older image tag, its original ports and " +
	"its intended mount. In the examples, databasus is the container name and <data-volume> the " +
	"name of the Docker volume; replace both with the values of your deployment. Stop a Compose " +
	"deployment first, run the cleanup for its storage, then recreate it with the edited image tag.\n" +
	"\n" +
	"Bind-mounted directory ./databasus-data:/databasus-data, run from the deployment directory " +
	"(use your path if another host directory is mounted):\n" +
	"  docker rm -f databasus\n" +
	"  sudo rm -rf -- ./databasus-data\n" +
	"Named Docker volume mounted at /databasus-data:\n" +
	"  docker rm -f databasus\n" +
	"  docker volume rm <data-volume>\n" +
	"No explicit mount, with the image's anonymous data volume:\n" +
	"  docker rm -f -v databasus\n" +
	"\n" +
	"docker rm -v deletes neither a bind-mounted directory nor a named volume. These examples cover " +
	"the internal database that runs inside the container. If DATABASE_DSN points to an external " +
	"database, deleting the Docker data does not reset it: configure a separate empty database for " +
	"the fresh installation."
