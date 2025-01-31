package ec2_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/service/ec2"
	sdkacctest "github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tfec2 "github.com/hashicorp/terraform-provider-aws/internal/service/ec2"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
)

func TestAccEC2EBSSnapshotCreateVolumePermission_basic(t *testing.T) {
	ctx := acctest.Context(t)
	resourceName := "aws_snapshot_create_volume_permission.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckAlternateAccount(t)
		},
		ErrorCheck:               acctest.ErrorCheck(t, ec2.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5FactoriesAlternate(ctx, t),
		CheckDestroy:             testAccCheckSnapshotCreateVolumePermissionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccEBSSnapshotCreateVolumePermissionConfig_basic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccSnapshotCreateVolumePermissionExists(ctx, resourceName),
					resource.TestCheckResourceAttrSet(resourceName, "account_id"),
					resource.TestCheckResourceAttrSet(resourceName, "snapshot_id"),
				),
			},
		},
	})
}

func TestAccEC2EBSSnapshotCreateVolumePermission_disappears(t *testing.T) {
	ctx := acctest.Context(t)
	resourceName := "aws_snapshot_create_volume_permission.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckAlternateAccount(t)
		},
		ErrorCheck:               acctest.ErrorCheck(t, ec2.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5FactoriesAlternate(ctx, t),
		CheckDestroy:             testAccCheckSnapshotCreateVolumePermissionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccEBSSnapshotCreateVolumePermissionConfig_basic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccSnapshotCreateVolumePermissionExists(ctx, resourceName),
					acctest.CheckResourceDisappears(ctx, acctest.Provider, tfec2.ResourceSnapshotCreateVolumePermission(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccEC2EBSSnapshotCreateVolumePermission_snapshotOwnerExpectError(t *testing.T) {
	ctx := acctest.Context(t)
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, ec2.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckSnapshotCreateVolumePermissionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config:      testAccEBSSnapshotCreateVolumePermissionConfig_snapshotOwner(rName),
				ExpectError: regexp.MustCompile(`owns EBS Snapshot`),
			},
		},
	})
}

func testAccCheckSnapshotCreateVolumePermissionDestroy(ctx context.Context) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Conn(ctx)

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_snapshot_create_volume_permission" {
				continue
			}

			snapshotID, accountID, err := tfec2.EBSSnapshotCreateVolumePermissionParseResourceID(rs.Primary.ID)

			if err != nil {
				return err
			}

			_, err = tfec2.FindCreateSnapshotCreateVolumePermissionByTwoPartKey(ctx, conn, snapshotID, accountID)

			if tfresource.NotFound(err) {
				continue
			}

			if err != nil {
				return err
			}

			return fmt.Errorf("EBS Snapshot CreateVolumePermission %s still exists", rs.Primary.ID)
		}

		return nil
	}
}

func testAccSnapshotCreateVolumePermissionExists(ctx context.Context, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No EBS Snapshot CreateVolumePermission ID is set")
		}

		snapshotID, accountID, err := tfec2.EBSSnapshotCreateVolumePermissionParseResourceID(rs.Primary.ID)

		if err != nil {
			return err
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Conn(ctx)

		_, err = tfec2.FindCreateSnapshotCreateVolumePermissionByTwoPartKey(ctx, conn, snapshotID, accountID)

		return err
	}
}

func testAccEBSSnapshotCreateVolumePermissionConfig_basic(rName string) string {
	return acctest.ConfigCompose(
		acctest.ConfigAlternateAccountProvider(),
		acctest.ConfigAvailableAZsNoOptIn(),
		fmt.Sprintf(`
resource "aws_ebs_volume" "test" {
  availability_zone = data.aws_availability_zones.available.names[0]
  size              = 1

  tags = {
    Name = %[1]q
  }
}

resource "aws_ebs_snapshot" "test" {
  volume_id = aws_ebs_volume.test.id

  tags = {
    Name = %[1]q
  }
}

data "aws_caller_identity" "test" {
  provider = "awsalternate"
}

resource "aws_snapshot_create_volume_permission" "test" {
  snapshot_id = aws_ebs_snapshot.test.id
  account_id  = data.aws_caller_identity.test.account_id
}
`, rName))
}

func testAccEBSSnapshotCreateVolumePermissionConfig_snapshotOwner(rName string) string {
	return acctest.ConfigCompose(acctest.ConfigAvailableAZsNoOptIn(),
		fmt.Sprintf(`
resource "aws_ebs_volume" "test" {
  availability_zone = data.aws_availability_zones.available.names[0]
  size              = 1

  tags = {
    Name = %[1]q
  }
}

resource "aws_ebs_snapshot" "test" {
  volume_id = aws_ebs_volume.test.id

  tags = {
    Name = %[1]q
  }
}

data "aws_caller_identity" "test" {}

resource "aws_snapshot_create_volume_permission" "test" {
  snapshot_id = aws_ebs_snapshot.test.id
  account_id  = data.aws_caller_identity.test.account_id
}
`, rName))
}
