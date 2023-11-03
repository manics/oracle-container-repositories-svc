See https://github.com/manics/binderhub-container-registry-helper/tree/main#readme

To build this repository locally with Podman:
`podman run -it --rm -v $PWD:/srv/jekyll:z -p4000:4000 -eJEKYLL_ROOTLESS=1 -ePAGES_REPO_NWO=manics/binderhub-container-registry-helper docker.io/jekyll/jekyll jekyll serve`
