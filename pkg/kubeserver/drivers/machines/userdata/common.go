package userdata

const (
	onecloudVMConfig = `
disable_swap() {
	swapoff -a
	# disable swap in fstab
	sed -i.bak -r 's/(.+ swap .+)/#\1/' /etc/fstab
}

get_master_address() {
	# Getting master ip from the metadata of the node. By default we try the public-ipv4
	# If we don't get any, we fall back to local-ipv4
	local master=""
	for i in $(seq 60); do
		echo "trying to get public-ipv4 $i / 60"
		master=$(curl --fail -s http://169.254.169.254/latest/meta-data/public-ipv4)
		if [[ $? == 0 ]] && [[ -n "$master" ]]; then
			break
		fi
		sleep 1
	done

	if [[ -z "$master" ]]; then
		echo "falling back to locak-ipv4"
		for i in $(seq 60); do
			echo "trying to get local-ipv4 $i / 60"
			master=$(curl --fail -s http://169.254.169.254/latest/meta-data/local-ipv4)
			if [[ $? == 0 ]] && [[ -n "$master" ]]; then
				break
			fi
			sleep 1
		done
	fi
	echo $master
}
`
)
