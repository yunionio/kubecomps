#!/usr/bin/env python3


import subprocess
import os
import urllib.request


def run_cmd(cmd):
    print(cmd)
    proc = subprocess.Popen(cmd, shell=True, universal_newlines=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT)
    while True:
        line = proc.stdout.readline()
        if not line:
            break
        print(line.rstrip())
    proc.wait()
    return proc.returncode

def run_cmds(cmds):
    for cmd in cmds:
        run_cmd(cmd)


class Image(object):

    def __init__(self, src_repo, src_name, tag, target_repo, target_name,
            arch=['amd64', 'arm64']):
        self._src_repo = src_repo
        self._src_name = src_name
        self._tag = tag
        self._target_repo = target_repo
        self._target_name = target_name
        self._arch = arch

    def get_src_path(self):
        return f'{self._src_repo}/{self._src_name}:{self._tag}'

    def get_target_path(self):
        return f'{self._target_repo}/{self._target_name}:{self._tag}'

    def get_target_arch_image(self, platform):
        target_img = f'{self.get_target_path()}-{platform}'
        return target_img

    def sync_image(self, platform):
        src_img = self.get_src_path()
        dst_img = f'{self.get_target_path()}-{platform}'
        target_img = self.get_target_arch_image(platform)
        if len(self._arch) == 1:
            src_img = src_img + '-' + self._arch[0]
            cmd = f'docker pull {src_img}'
        else:
            cmd = f'docker pull {src_img} --platform {platform}'
        cmds = [cmd,
                f'docker tag {src_img} {dst_img}',
                f'docker push {dst_img}']
        run_cmds(cmds)

    def sync_archs_image(self):
        for p in self._arch:
            self.sync_image(p)
        if len(self._arch) == 1:
            return
        cmds = []
        m_create_cmd = f'docker manifest create {self.get_target_path()} ' + ' '.join([self.get_target_arch_image(x) for x in self._arch])
        cmds.append(m_create_cmd)
        for a in self._arch:
            c = f'docker manifest annotate {self.get_target_path()} {self.get_target_arch_image(a)} --arch {a}'
            cmds.append(c)
        cmds.append(f'docker manifest push {self.get_target_path()}')
        run_cmds(cmds)


class DownloadFile(object):

    def __init__(self, url, dest_dir):
        self._url = url
        self._dest_dir = dest_dir

    def get_name(self):
        return os.path.basename(self._url)

    def get_target_dir(self, output_dir):
        raise Exception("Not ipml")

    def save_archive(self):
        dest_dir = self.get_target_dir(self._dest_dir)
        if not os.path.exists(dest_dir):
            os.makedirs(dest_dir)
        url = self._url
        target_path = os.path.join(dest_dir, self.get_name())
        print(f"Downloading {url} to {target_path}")
        urllib.request.urlretrieve(url, target_path)


class DownloadGithubFile(DownloadFile):

    def __init__(self, dest_dir, repo, version, arch):
        self._version = version
        self._arch = arch
        self._repo = repo
        super(DownloadGithubFile, self).__init__(self.get_url(), dest_dir)

    def get_url(self):
        return f'https://github.com/{self._repo}/' + self.get_target_filepath()

    def get_target_dir(self, output_dir):
        basedir = os.path.dirname(self.get_target_filepath())
        output_dir = os.path.join(output_dir, basedir)
        return output_dir


class DownloadK8sFile(DownloadFile):

    def __init__(self, dest_dir, version, arch, bin):
        self._version = version
        self._arch = arch
        self._bin = bin
        super().__init__(self.get_url(), dest_dir)

    def get_url(self):
        return f'https://storage.googleapis.com/kubernetes-release/release/{self._version}/bin/linux/{self._arch}/{self._bin}'

    def get_target_filepath(self):
        return f'kubernetes-release/release/{self._version}/bin/linux/{self._arch}/{self._bin}'

    def get_target_dir(self, output_dir):
        basedir = os.path.dirname(self.get_target_filepath())
        output_dir = os.path.join(output_dir, basedir)
        return output_dir


class DownloadKubelet(DownloadK8sFile):

    def __init__(self, dest_dir, version, arch,):
        super().__init__(dest_dir, version, arch, 'kubelet')


class DownloadKubeadm(DownloadK8sFile):

    def __init__(self, dest_dir, version, arch,):
        super().__init__(dest_dir, version, arch, 'kubeadm')


class DownloadKubectl(DownloadK8sFile):

    def __init__(self, dest_dir, version, arch,):
        super().__init__(dest_dir, version, arch, 'kubectl')


class DownloadCalicoctl(DownloadGithubFile):

    def __init__(self, dest_dir, version, arch):
        super(DownloadCalicoctl, self).__init__(dest_dir, 'projectcalico', version, arch)

    def get_target_filepath(self):
        return f'calicoctl/releases/download/{self._version}/calicoctl-linux-{self._arch}'


class DownloadCrictl(DownloadGithubFile):

    def __init__(self, dest_dir, version, arch):
        super(DownloadCrictl, self).__init__(dest_dir, 'kubernetes-sigs', version, arch)

    def get_target_filepath(self):
        return f'cri-tools/releases/download/{self._version}/crictl-{self._version}-linux-{self._arch}.tar.gz'


class DownloadCNI(DownloadGithubFile):

    def __init__(self, dest_dir, version, arch):
        super().__init__(dest_dir, 'containernetworking', version, arch)

    def get_target_filepath(self):
        return f'plugins/releases/download/{self._version}/cni-plugins-linux-{self._arch}-{self._version}.tgz'


class DownloadCalicoCRDs(DownloadGithubFile):

    def __init__(self, dest_dir, version):
        super().__init__(dest_dir, 'projectcalico', version, '')

    def get_target_filepath(self):
        return f'calico/archive/{self._version}.tar.gz'


def docker_pull_push(src, target_repo):
    target = os.path.join(target_repo, src.split('/')[-1])
    cmds = [
        f'docker pull {src}',
        f'docker tag {src} {target}',
        f'docker push {target}',
    ]
    run_cmds(cmds)


def docker_cluster_proportional_image(taget_repo):
    docker_pull_push('k8s.gcr.io/cpa/cluster-proportional-autoscaler-arm64:1.8.5', taget_repo)
    docker_pull_push('k8s.gcr.io/cpa/cluster-proportional-autoscaler-amd64:1.8.5', taget_repo)


def sync_images(repo):
    imgs = [
        Image("docker.io/library", "nginx", "1.19", repo, "nginx"),
        # Image("calico", "node", "v3.19.3", repo, "calico-node"),
        # Image("calico", "cni", "v3.19.3", repo, "calico-cni"),
        # Image("calico", "kube-controllers", "v3.19.3", repo, "calico-kube-controllers"),
        # Image("calico", "typha", "v3.19.3", repo, "calico-typha"),
        # Image('quay.io/coreos', 'etcd', 'v3.4.13', repo, 'etcd', arch=['arm64']),
        # Image("k8s.gcr.io/dns", "k8s-dns-node-cache", "1.16.0", repo, "k8s-dns-node-cache"),
        # Image("k8s.grc.io/cpa", "cluster-proportional-autoscaler", "1.8.3", repo, "cluster-proportional-autoscaler"),
    ]
    for i in imgs:
        i.sync_archs_image()


def download_files():
    output_dir = "./_output/binaries"

    def k8s_bins(version):
        return [
            DownloadKubectl(output_dir, version, 'amd64'),
            DownloadKubectl(output_dir, version, 'arm64'),
            DownloadKubeadm(output_dir, version, 'amd64'),
            DownloadKubeadm(output_dir, version, 'arm64'),
            DownloadKubelet(output_dir, version, 'amd64'),
            DownloadKubelet(output_dir, version, 'arm64'),
        ]

    fs = [
        # DownloadCalicoctl(output_dir, "v3.19.3", "amd64"),
        # DownloadCalicoctl(output_dir, "v3.19.3", "arm64"),
        # DownloadCrictl(output_dir, 'v1.20.0', "amd64"),
        # DownloadCrictl(output_dir, 'v1.20.0', "arm64"),
        # DownloadCNI(output_dir, 'v0.9.1', 'arm64'),
        # DownloadCNI(output_dir, 'v0.9.1', 'amd64'),
        # DownloadCNI(output_dir, 'v0.8.6', 'arm64'),
        # DownloadCNI(output_dir, 'v0.8.6', 'amd64'),
        DownloadCalicoCRDs(output_dir, 'v3.19.2'),
    ]
    # fs.extend(k8s_bins('v1.17.0'))
    # fs.extend(k8s_bins('v1.20.0'))
    for f in fs:
        f.save_archive()


if __name__ == '__main__':
    repo = 'registry.cn-beijing.aliyuncs.com/yunionio'
    sync_images(repo)
    #docker_cluster_proportional_image(repo)
    # download_files()
