import json
from tornado import httpclient
from traitlets import Unicode
from binderhub.registry import DockerRegistry


class ExternalRegistryHelper(DockerRegistry):

    service_url = Unicode(
        "http://binderhub-container-registry-helper:8080",
        allow_none=False,
        help="The URL of the registry helper micro-service.",
        config=True,
    )

    auth_token = Unicode(
        "secret-token",
        help="The auth token to use when accessing the registry helper micro-service.",
        config=True,
    )

    async def _request(self, endpoint, **kwargs):
        client = httpclient.AsyncHTTPClient()
        repo_url = f"{self.service_url}{endpoint}"
        headers = {"Authorization": f"Bearer {self.auth_token}"}
        repo = await client.fetch(repo_url, headers=headers, **kwargs)
        return json.loads(repo.body.decode("utf-8"))

    async def _get_image(self, image, tag):
        repo_url = f"/image/{image}:{tag}"
        self.log.debug(f"Checking whether image exists: {repo_url}")
        try:
            image_json = await self._request(repo_url)
            return image_json
        except httpclient.HTTPError as e:
            if e.code == 404:
                return None
            else:
                raise

    async def get_image_manifest(self, image, tag):
        """
        Checks whether the image exists in the registry.

        If the container repository doesn't exist create the repository.

        The container repository name may not be the same as the BinderHub image name.

        E.g. Oracle Container Registry (OCIR) has the form:
        OCIR_NAMESPACE/OCIR_REPOSITORY_NAME:TAG

        These extra components are handled automatically by the registry helper
        so BinderHub repository names such as OCIR_NAMESPACE/OCIR_REPOSITORY_NAME
        can be used directly, it is not necessary to remove the extra components.

        Returns the image manifest if the image exists, otherwise None
        """

        repo_url = f"/repo/{image}"
        self.log.debug(f"Checking whether repository exists: {repo_url}")
        try:
            repo_json = await self._request(repo_url)
            repo_exists = repo_json
        except httpclient.HTTPError as e:
            if e.code == 404:
                repo_exists = False
            else:
                raise

        if repo_exists:
            return await self._get_image(image, tag)
        else:
            self.log.debug(f"Creating repository: {repo_url}")
            await self._request(repo_url, method="POST", body="")
            return None


c.BinderHub.registry_class = ExternalRegistryHelper
