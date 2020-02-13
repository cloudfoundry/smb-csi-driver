package nodeserver

import (
	"code.cloudfoundry.org/goshims/execshim"
	"code.cloudfoundry.org/goshims/osshim"
	"context"
	"fmt"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"os"
)

var errorFmt = "Error: a required property [%s] was not provided"

type smbNodeServer struct {
	execshim execshim.Exec
	osshim osshim.Os
}

func NewNodeServer(execshim execshim.Exec, osshim osshim.Os) csi.NodeServer {
	return &smbNodeServer{
		execshim, osshim,
	}
}

func (smbNodeServer) NodeGetCapabilities(context.Context, *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	return &csi.NodeGetCapabilitiesResponse{}, nil
}

func (smbNodeServer) NodeStageVolume(context.Context, *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	panic("implement me")
}

func (smbNodeServer) NodeUnstageVolume(context.Context, *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	panic("implement me")
}

func (n smbNodeServer) NodePublishVolume(c context.Context, r *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	if r.VolumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf(errorFmt, "VolumeCapability"))
	}

	err := os.MkdirAll(r.TargetPath, os.ModePerm)
	if err != nil {
		println(err.Error())
	}
	share := r.GetVolumeContext()["share"]
	username := r.GetVolumeContext()["username"]
	password := r.GetVolumeContext()["password"]

	log.Printf("local target path: %s", r.TargetPath)

	mountOptions := fmt.Sprintf("username=%s,password=%s", username, password)

	cmdshim := n.execshim.Command("mount", "-t", "cifs", "-o", mountOptions, share, r.TargetPath)
	err = cmdshim.Start()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	fmt.Println(fmt.Sprintf("started mount to %s", share))

	err = cmdshim.Wait()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	fmt.Println(fmt.Sprintf("finished mount to %s", share))

	return &csi.NodePublishVolumeResponse{}, nil
}

func (n smbNodeServer) NodeUnpublishVolume(c context.Context, r *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	if r.TargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf(errorFmt, "TargetPath"))
	}

	log.Printf("about to remove dir")

	cmdshim := n.execshim.Command("umount", r.TargetPath)
	err := cmdshim.Start()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	log.Print("started umount")

	err = cmdshim.Wait()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	log.Printf("finished umount")

	err = n.osshim.Remove(r.TargetPath)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	log.Printf("removed dir")

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (smbNodeServer) NodeGetVolumeStats(context.Context, *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	panic("implement me")
}

func (smbNodeServer) NodeExpandVolume(context.Context, *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	panic("implement me")
}

func (s smbNodeServer) NodeGetInfo(context.Context, *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	nodeId, err := s.osshim.Hostname()
	if err != nil {
		return nil, err
	}

	return &csi.NodeGetInfoResponse{
		NodeId: nodeId,
	}, nil
}