package kubespray

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"yunion.io/x/pkg/errors"
)

func TestCommonVars(t *testing.T) {
	Convey("Validate CommonVars validate", t, func() {

		newKVar := func(kv string) error {
			return ValidateKubernetesVersion(kv)
		}

		Convey("Check KubernetesVersion", func() {

			Convey("Empty version should be invalid", func() {
				So(newKVar(""), ShouldBeError, ErrKubernetesVersionEmpty)
			})

			for _, invalidV := range []string{
				"v1.1.0",
				"2.1.0",
				"1.123.0",
				"test",
			} {
				Convey(fmt.Sprintf("The %q should be invalid", invalidV), func() {
					So(newKVar(invalidV), ShouldBeError, errors.Wrapf(ErrKubernetesVersionInvalidFormat, "%s", invalidV))
				})

			}

			for _, validV := range []string{
				"1.1.0",
				"1.12.0",
				"1.16.9",
			} {
				Convey(fmt.Sprintf("The %q should valid", validV), func() {
					So(newKVar(validV), ShouldBeNil)
				})
			}

		})
	})
}
