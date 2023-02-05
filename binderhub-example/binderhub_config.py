import json
from tornado import httpclient
from traitlets import Unicode
from binderhub.registry import DockerRegistry


class ExternalRegistryHelper(DockerRegistry):

    service_url = Unicode(
        "http://oracle-container-repositories-svc:8080",
        allow_none=False,
        help="The URL of the registry helper micro-service.",
        config=True,
    )

    auth_token = Unicode(
        "secret-token",
        help="The auth token to use when accessing the registry helper micro-service.",
        config=True,
    )

    async def get_image_manifest(self, image, tag):
        """
        If the container repository exists use the standard Docker Registry API
        to check for the image tag.
        Otherwise create the container repository.

        The full registry image URL has the form:
        CONTAINER_REGISTRY/OCIR_NAMESPACE/OCIR_REPOSITORY_NAME:TAG
        the BinderHub image is OCIR_NAMESPACE/OCIR_REPOSITORY_NAME
        """
        client = httpclient.AsyncHTTPClient()
        repo_url = f"{self.service_url}/repo/{image}"
        headers = {"Authorization": f"Bearer {self.auth_token}"}

        self.log.debug(f"Checking whether repository exists: {repo_url}")
        try:
            repo = await client.fetch(repo_url, headers=headers)
            repo_exists = True
        except httpclient.HTTPError as e:
            if e.code == 404:
                repo_exists = False
            else:
                raise

        if repo_exists:
            repo_json = json.loads(repo.body.decode("utf-8"))
            self.log.debug(f"Repository exists: {repo_json}")
            return await super().get_image_manifest(image, tag)
        else:
            self.log.debug(f"Creating repository: {repo_url}")
            await client.fetch(repo_url, headers=headers, method="POST", body="")
            return None


c.BinderHub.registry_class = ExternalRegistryHelper
