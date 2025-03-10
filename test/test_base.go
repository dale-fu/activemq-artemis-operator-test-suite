package test

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	"github.com/onsi/gomega"
	"github.com/rh-messaging/shipshape/pkg/framework"
	"github.com/rh-messaging/shipshape/pkg/framework/ginkgowrapper"
	"github.com/rh-messaging/shipshape/pkg/framework/log"
	"k8s.io/klog"
)

// PrepareNamespace once this file is imported, the "init()" method will be called automatically
// by Ginkgo and so, within your test suites you have to explicitly invoke this method
// as it will run your specs and setup the appropriate reporters (if any requested).
// This method MUST be called (otherwise the init() might not be executed).
// The uniqueId is used to help composing the generated JUnit file name (when --report-dir
// is specified when running your test).

var (
	Config = TestConfiguration{
		"registry.redhat.io/amq7/amq-broker-rhel7-operator:latest",
		"registry.redhat.io/amq7/amq-broker:latest", "registry.redhat.io/amq7/amq-broker:latest",
		"amq-broker", // BrokerName
		"",           //Repository path
		true,         // DebugRun
		false,        // Non-admin run
		false,        // Using latest (known) CR
		false,        // IBMz
		false,        // PPC
		false,        // Openshift
		1,            // TimeoutMultiplier
		false}        // GlobalOperator

)

type TestConfiguration struct {
	OperatorImageName string
	BrokerImageName   string
	BrokerImageOther  string
	BrokerName        string
	RepositoryPath    string
	DebugRun          bool
	AdminUnavailable  bool
	NeedsLatestCR     bool
	IBMz              bool
	PPC               bool
	Openshift         bool
	TimeoutMultiplier int
	GlobalOperator    bool
}

const (
	Username       = "admin"
	Password       = "admin"
	ProjectRootDir = "activemq-artemis-operator-test-suite"
)

var MainYamls = []string{
	"service_account.yaml",
	"operator.yaml",
}

var LocalYamls = []string{
	"role.yaml",
	"role_binding.yaml",
}

var GlobalYamls = []string{
	"cluster_role.yaml",
	"cluster_role_binding.yaml",
}

var CrdsV1 = []string{
	"crds/broker_v2alpha1_activemqartemis_crd.yaml",
	"crds/broker_v2alpha1_activemqartemisaddress_crd.yaml",
	"crds/broker_v2alpha1_activemqartemisscaledown_crd.yaml",
}

var CrdsV2 = []string{
	"crds/broker_activemqartemis_crd.yaml",
	"crds/broker_activemqartemisaddress_crd.yaml",
	"crds/broker_activemqartemisscaledown_crd.yaml",
	"crds/broker_activemqartemissecurity_crd.yaml",
}

func loadFromSlice(slice []string, path string) ([][]byte, error) {
	var result [][]byte

	for _, item := range slice {
		byteItem, err := ioutil.ReadFile(path + "/" + item)
		if err != nil {
			return nil, err
		} else {
			result = append(result, byteItem)
		}
	}
	return result, nil
}

func LoadYamls(path string) ([][]byte, error) {
	//Load all the main stuff
	var result [][]byte
	loaded, err := loadFromSlice(MainYamls, path)
	if err != nil {
		return nil, err
	} else {
		for _, item := range loaded {
			result = append(result, item)
		}
	}
	loaded, _ = loadFromSlice(LocalYamls, path)
	if Config.GlobalOperator {
		log.Logf("Adding global yamls")
		loadedNew, _ := loadFromSlice(GlobalYamls, path)
		for _, item := range loadedNew {
			loaded = append(loaded, item)
		}
	}
	for _, item := range loaded {
		result = append(result, item)
	}

	//And all the other stuff.
	if Config.NeedsLatestCR {
		loaded, err := loadFromSlice(CrdsV2, path)
		if err != nil {
			return nil, err
		} else {
			for _, item := range loaded {
				result = append(result, item)
			}
		}
	} else {
		loaded, err := loadFromSlice(CrdsV1, path)
		if err != nil {
			return nil, err
		} else {
			for _, item := range loaded {
				result = append(result, item)
			}
		}
	}
	return result, nil
}

//func init() {
func RegisterFlags() {
	// Defaulting to latest released broker image
	// Needs authentication with registry.redhat.io!
	loadConfig()
	//  Default OperatorImage is provided by shipshape.
	flag.StringVar(&Config.OperatorImageName, "operator-image", Config.OperatorImageName, "operator image url")
	flag.StringVar(&Config.BrokerImageName, "broker-image", Config.BrokerImageName, "broker image url")
	flag.StringVar(&Config.BrokerImageOther, "broker-image-second", Config.BrokerImageOther, "broker image url to update to")
	flag.StringVar(&Config.BrokerName, "broker-name", Config.BrokerName, "broker name")
	flag.BoolVar(&Config.DebugRun, "debug-run", false, "debug run toggle")
	flag.StringVar(&Config.RepositoryPath, "repository", Config.RepositoryPath, "path to the amq operator deployment repository")
	flag.BoolVar(&Config.AdminUnavailable, "no-admin-available", false, "sets cluster-wide admin privileges availability")
	flag.BoolVar(&Config.NeedsLatestCR, "v2", false, "defines if V2 version of the API needs to be used")
	flag.BoolVar(&Config.IBMz, "ibmz", false, "defines if shipshape should use ibmz client images")
	flag.BoolVar(&Config.PPC, "ppc", false, "defines if shipshape should use ppc64le client images")
	flag.BoolVar(&Config.Openshift, "openshift", false, "defines if shipshape should use openshift specific APIs")
	flag.IntVar(&Config.TimeoutMultiplier, "timeoutMult", 1, "defines timeout multiplier")
	flag.BoolVar(&Config.GlobalOperator, "global", false, "defines if operator is expected to be globally installed (managing multiple namespaces)")

}

func loadConfig() {
	cwd := getProjectRootPath()
	log.Logf("yaml loading from: " + cwd)
	yamlFile, err := ioutil.ReadFile(cwd + "/" + "config.yaml")
	if err != nil {
		log.Logf("yaml load err: #%v", err)
	} else {
		err = yaml.Unmarshal(yamlFile, &Config)
		if err != nil {
			log.Logf("yaml parsing err: #%v", err)
		}
	}
}

func Initialize(m *testing.M) {
	log.Logf("Initializeing")
	framework.RegisterFlags()
	RegisterFlags()
	flag.Parse()
	log.Logf("Value: %v", Config.NeedsLatestCR)
	gomega.RegisterFailHandler(ginkgowrapper.Fail)
	if m != nil {
		m.Run()
	}
}

func PrepareNamespace(t *testing.T, uniqueId string, description string) {
	// If any ginkgoReporter has been defined, use them.
	if framework.TestContext.ReportDir != "" {
		ginkgo.RunSpecsWithDefaultAndCustomReporters(t, description, generateReporter(uniqueId))
	} else {
		ginkgo.RunSpecs(t, description)
	}
}

// generateReporter returns a slice of ginkgo.Reporter if reportDir has been provided
func generateReporter(uniqueId string) []ginkgo.Reporter {
	var ginkgoReporters []ginkgo.Reporter
	// If report dir specified, create it
	if framework.TestContext.ReportDir != "" {
		if err := os.MkdirAll(framework.TestContext.ReportDir, 0755); err != nil {
			klog.Errorf("Failed creating report directory: %v", err)
		} else {
			ginkgoReporters = append(ginkgoReporters, reporters.NewJUnitReporter(
				path.Join(framework.TestContext.ReportDir,
					fmt.Sprintf("junit_%v%s%02d.xml",
						framework.TestContext.ReportPrefix,
						uniqueId,
						config.GinkgoConfig.ParallelNode))))
		}
	}

	return ginkgoReporters
}

// Before suite validation setup (happens only once per test suite)
var _ = ginkgo.SynchronizedBeforeSuite(func() []byte {
	// Unique initialization (node 1 only)
	return nil
}, func(data []byte) {
	// Initialization for each parallel node
}, 10)

// After suite validation teardown (happens only once per test suite)
var _ = ginkgo.SynchronizedAfterSuite(func() {
	// All nodes tear down
}, func() {
	// Node1 only tear down
	if !Config.DebugRun {
		framework.RunCleanupActions(framework.AfterEach)
		framework.RunCleanupActions(framework.AfterSuite)
	}
}, 10)

func getProjectRootPath() string {
	cwd, err := os.Getwd()
	cwdOrig := cwd
	if err != nil {
		panic(err)
	}
	for {
		if strings.HasSuffix(cwd, "/"+ProjectRootDir) {
			return cwd
		}
		lastSlashIndex := strings.LastIndex(cwd, "/")
		if lastSlashIndex == -1 {
			panic(cwdOrig + " did not contain /" + ProjectRootDir)
		}
		cwd = cwd[0:lastSlashIndex]
	}
}
