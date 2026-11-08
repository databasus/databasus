import os
import uuid
from dataclasses import dataclass

from database_server import (
    BACKED_UP_DATABASE,
    SERVER_HOST_ON_COMPOSE_NETWORK,
    SERVER_PORT,
    SERVER_USER,
)
from navigation_api import NavigationRequest, get_identity, request_navigation_api
from playwright.sync_api import Page


@dataclass(frozen=True)
class NavigationWorkspace:
    workspace_id: str
    database_id: str
    database_name: str

    def get_database_path(self, tab: str = "config") -> str:
        return f"/workspaces/{self.workspace_id}/databases/{self.database_id}/{tab}"


def create_navigation_workspace(page: Page) -> NavigationWorkspace:
    workspace = request_navigation_api(
        page, NavigationRequest("/workspaces", {"name": f"URL navigation {uuid.uuid4()}"})
    )
    workspace_id = get_identity(workspace)
    database_name = f"Linked database {uuid.uuid4()}"
    database = request_navigation_api(
        page,
        NavigationRequest(
            "/databases",
            {
                "workspaceId": workspace_id,
                "name": database_name,
                "notifierIds": [],
                "settings": {
                    "type": "POSTGRES_LOGICAL",
                    "host": SERVER_HOST_ON_COMPOSE_NETWORK,
                    "port": int(SERVER_PORT),
                    "username": SERVER_USER,
                    "password": os.environ["E2E_DATABASE_PASSWORD"],
                    "database": BACKED_UP_DATABASE,
                    "sslMode": "disable",
                    "sshTunnel": {"isEnabled": False},
                    "includeSchemas": [],
                    "excludeTables": [],
                    "excludeTableData": [],
                    "cpuCount": 1,
                    "isSkipUserMappings": False,
                },
            },
        ),
    )

    return NavigationWorkspace(workspace_id, get_identity(database), database_name)
