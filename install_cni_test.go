package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"os"
	"os/exec"
	"io/ioutil"
)

var script = "k8s-install/scripts/install-cni.sh"

var defaultEnvVars = map[string]string {
	"KUBERNETES_SERVICE_HOST": "127.0.0.1",
	"KUBERNETES_SERVICE_PORT": "8080",
	"KUBERNETES_NODE_NAME": "k8s-node-01",
	"SERVICEACCOUNT_TOKEN": "my_service_token",
}

// runCniScript will run install-cni.sh, and return the output of the command and an error
func runCniScript() (string, error) {
	result, err := exec.Command(script).CombinedOutput()
	if err != nil {
		GinkgoWriter.Write(result)
		return string(result), err
	}
	return string(result), nil
}

var _ = BeforeSuite(func() {
	os.Setenv("SLEEP", "false")
})

var _ = Describe("install-cni.sh tests", func() {
	BeforeEach(func() {
		// Create the needed dirs.
		os.MkdirAll("/host/opt/cni/bin", 755)
		os.MkdirAll("/host/etc/cni/net.d", 755)

		// Setup out env vars.
		for k, v := range defaultEnvVars {
			os.Setenv(k, v)
		}

		// Move the temp config file into place.
		exec.Command("cp", "k8s-install/scripts/calico.conf.default", "/calico.conf.tmp").Run()
	})

	AfterEach(func() {
		// Unset our env vars
		for k, _ := range defaultEnvVars {
			os.Unsetenv(k)
		}
		// Remove the bins and conf file between each run.
		exec.Command("rm", "/host/opt/cni/bin/*").Run()
		exec.Command("rm", "/host/etc/net.d/*").Run()
	})

	Describe("Run install-cni", func() {
		Context("With default values", func() {
			It("Should install bins and config", func() {
				_, err := runCniScript()
				Expect(err).NotTo(HaveOccurred())

				// Get a list of files in the defualt CNI bin location.
				files, _ := ioutil.ReadDir("/host/opt/cni/bin")
				names := []string{}
				for _, file := range files {
					names = append(names, file.Name())
				}

				// Get a list of files in the default location for CNI config.
				files, _ = ioutil.ReadDir("/host/etc/cni/net.d")
				for _, file := range files {
					names = append(names, file.Name())
				}

				Expect(names).To(ContainElement("calico"))
				Expect(names).To(ContainElement("calico-ipam"))
				Expect(names).To(ContainElement("10-calico.conf"))
			})
			It("Should parse and output a templated config", func() {
				_, err := runCniScript()
				Expect(err).NotTo(HaveOccurred())

				var expected = []byte{}
				if file, err := os.Open("test_utils/expected_10-calico.conf"); err != nil {
					Fail("Could not open test_utils/expected_10-calico.conf for reading")
				} else {
					file.Read(expected)
					file.Close()
				}

				var received = []byte{}
				if file, err := os.Open("/host/etc/cni/net.d/10-calico.conf"); err != nil {
					Fail("Could not open /host/etc/cni/net.d/10-calico.conf for reading")
				} else {
					file.Read(received)
					file.Close()
				}

				Expect(expected).To(Equal(received))
			})
		})
		Context("With modified env vars", func() {
			It("Should rename '10-calico.conf' to '10-test.conf'", func() {
				os.Setenv("CNI_CONF_NAME", "10-test.conf")
				_, err := runCniScript()
				os.Unsetenv("CNI_CONF_NAME")
				Expect(err).NotTo(HaveOccurred())

				// Check to see if the file was named correctly.
				if _, err := os.Stat("/host/etc/cni/net.d/10-test.conf"); os.IsNotExist(err) {
					Fail("Could not locate /host/etc/cni/net.d/10-test.conf")
				}
			})
		})
		Context("When it cannot write CNI bins", func() {
			It("Should skip non-existent directory", func() {
				exec.Command("rm", "-rf", "/host/opt/cni/bin").Run()
				result, err := runCniScript()

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(ContainSubstring("/host/opt/cni/bin is non-writeable, skipping"))
			})
			It("Should fail when we cannot write the CNI config", func() {
				exec.Command("rm", "-rf", "/host/etc/cni/net.d").Run()
				_, err := runCniScript()

				Expect(err).To(HaveOccurred())
			})
		})
	})
})