// Copyright DataStax, Inc.
// Please see the included license file for details.

package v1beta1

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_makeImage(t *testing.T) {
	type args struct {
		serverType    string
		serverImage   string
		serverVersion string
	}
	tests := []struct {
		name      string
		args      args
		want      string
		errString string
	}{
		{
			name: "test empty image",
			args: args{
				serverImage:   "",
				serverType:    "dse",
				serverVersion: "6.8.0",
			},
			want:      "datastax/dse-server:6.8.0",
			errString: "",
		},
		{
			name: "test empty image cassandra",
			args: args{
				serverImage:   "",
				serverType:    "cassandra",
				serverVersion: "3.11.6",
			},
			want:      "datastax/cassandra-mgmtapi-3_11_6:v0.1.5",
			errString: "",
		},
		{
			name: "test private repo server",
			args: args{
				serverImage:   "datastax.jfrog.io/secret-debug-image/dse-server:6.8.0-test123",
				serverType:    "dse",
				serverVersion: "6.8.0",
			},
			want:      "datastax.jfrog.io/secret-debug-image/dse-server:6.8.0-test123",
			errString: "",
		},
		{
			name: "test unknown version",
			args: args{
				serverImage:   "",
				serverType:    "dse",
				serverVersion: "6.7.0",
			},
			want:      "",
			errString: "server 'dse' and version '6.7.0' do not work together",
		},
		{
			name: "test 6.8.1",
			args: args{
				serverImage:   "",
				serverType:    "dse",
				serverVersion: "6.8.1",
			},
			want:      "datastax/dse-server:6.8.1",
			errString: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := makeImage(tt.args.serverType, tt.args.serverVersion, tt.args.serverImage)
			if got != tt.want {
				t.Errorf("makeImage() = %v, want %v", got, tt.want)
			}
			if err == nil {
				if tt.errString != "" {
					t.Errorf("makeImage() err = %v, want %v", err, tt.errString)
				}
			} else {
				if err.Error() != tt.errString {
					t.Errorf("makeImage() err = %v, want %v", err, tt.errString)
				}
			}
		})
	}
}

func TestCassandraDatacenter_GetServerImage(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       CassandraDatacenterSpec
		Status     CassandraDatacenterStatus
	}
	tests := []struct {
		name      string
		fields    fields
		want      string
		errString string
	}{
		{
			name: "explicit server image specified",
			fields: fields{
				Spec: CassandraDatacenterSpec{
					ServerImage:   "jfrog.io:6789/dse-server-team/dse-server:6.8.0-123",
					ServerVersion: "6.8.0",
				},
			},
			want:      "jfrog.io:6789/dse-server-team/dse-server:6.8.0-123",
			errString: "",
		},
		{
			name: "invalid version specified",
			fields: fields{
				Spec: CassandraDatacenterSpec{
					ServerImage:   "",
					ServerVersion: "9000",
				},
			},
			want:      "",
			errString: "server '' and version '9000' do not work together",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dc := &CassandraDatacenter{
				TypeMeta:   tt.fields.TypeMeta,
				ObjectMeta: tt.fields.ObjectMeta,
				Spec:       tt.fields.Spec,
				Status:     tt.fields.Status,
			}
			got, err := dc.GetServerImage()
			if got != tt.want {
				t.Errorf("CassandraDatacenter.GetServerImage() = %v, want %v", got, tt.want)
			}
			if err == nil {
				if tt.errString != "" {
					t.Errorf("CassandraDatacenter.GetServerImage() err = %v, want %v", err, tt.errString)
				}
			} else {
				if err.Error() != tt.errString {
					t.Errorf("CassandraDatacenter.GetServerImage() err = %v, want %v", err, tt.errString)
				}
			}

		})
	}
}

func Test_GenerateBaseConfigString(t *testing.T) {
	tests := []struct {
		name      string
		dc        *CassandraDatacenter
		want      string
		errString string
	}{
		{
			name: "Simple Test",
			dc: &CassandraDatacenter{
				ObjectMeta: metav1.ObjectMeta{
					Name: "exampleDC",
				},
				Spec: CassandraDatacenterSpec{
					ClusterName: "exampleCluster",
					Config:      []byte("{\"cassandra-yaml\":{\"authenticator\":\"AllowAllAuthenticator\",\"batch_size_fail_threshold_in_kb\":1280}}"),
				},
			},
			want:      `{"cassandra-yaml":{"authenticator":"AllowAllAuthenticator","batch_size_fail_threshold_in_kb":1280},"cluster-info":{"name":"exampleCluster","seeds":"exampleCluster-seed-service"},"datacenter-info":{"graph-enabled":0,"name":"exampleDC","solr-enabled":0,"spark-enabled":0}}`,
			errString: "",
		},
		{
			name: "Simple Test for error",
			dc: &CassandraDatacenter{
				ObjectMeta: metav1.ObjectMeta{
					Name: "exampleDC",
				},
				Spec: CassandraDatacenterSpec{
					ClusterName: "exampleCluster",
					Config:      []byte("\"cassandra-yaml\":{\"authenticator\":\"AllowAllAuthenticator\",\"batch_size_fail_threshold_in_kb\":1280}}"),
				},
			},
			want:      "",
			errString: "Error parsing Spec.Config for CassandraDatacenter resource: invalid character ':' after top-level value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.dc.GetConfigAsJSON()
			if got != tt.want {
				t.Errorf("GenerateBaseConfigString() got = %v, want %v", got, tt.want)
			}
			if err == nil {
				if tt.errString != "" {
					t.Errorf("GenerateBaseConfigString() err = %v, want %v", err, tt.errString)
				}
			} else {
				if err.Error() != tt.errString {
					t.Errorf("GenerateBaseConfigString() err = %v, want %v", err, tt.errString)
				}
			}
		})
	}
}

func TestCassandraDatacenter_GetContainerPorts(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       CassandraDatacenterSpec
		Status     CassandraDatacenterStatus
	}
	tests := []struct {
		name    string
		fields  fields
		want    []corev1.ContainerPort
		wantErr bool
	}{
		{
			name: "Happy Path",
			fields: fields{
				Spec: CassandraDatacenterSpec{
					ClusterName: "exampleCluster",
					Config:      []byte("{\"cassandra-yaml\":{\"authenticator\":\"AllowAllAuthenticator\",\"batch_size_fail_threshold_in_kb\":1280}}"),
				},
			},
			want: []corev1.ContainerPort{
				{
					Name:          "native",
					ContainerPort: 9042,
				}, {
					Name:          "inter-node-msg",
					ContainerPort: 8609,
				}, {
					Name:          "intra-node",
					ContainerPort: 7000,
				}, {
					Name:          "tls-intra-node",
					ContainerPort: 7001,
				}, {
					Name:          "mgmt-api-http",
					ContainerPort: 8080,
				}},
			wantErr: false,
		},
		{
			name: "Expose Prometheus",
			fields: fields{
				Spec: CassandraDatacenterSpec{
					ClusterName: "exampleCluster",
					Config:      []byte("{\"cassandra-yaml\":{\"10-write-prom-conf\":{\"enabled\":true,\"port\":9103,\"staleness-delta\":300},\"authenticator\":\"AllowAllAuthenticator\",\"batch_size_fail_threshold_in_kb\":1280}}"),
				},
			},
			want: []corev1.ContainerPort{
				{
					Name:          "native",
					ContainerPort: 9042,
				}, {
					Name:          "inter-node-msg",
					ContainerPort: 8609,
				}, {
					Name:          "intra-node",
					ContainerPort: 7000,
				}, {
					Name:          "tls-intra-node",
					ContainerPort: 7001,
				}, {
					Name:          "mgmt-api-http",
					ContainerPort: 8080,
				}, {
					Name:          "prometheus",
					ContainerPort: 9103,
				}},
			wantErr: false,
		},
		{
			name: "Expose Prometheus - No config",
			fields: fields{
				Spec: CassandraDatacenterSpec{
					ClusterName: "exampleCluster",
				},
			},
			want: []corev1.ContainerPort{
				{
					Name:          "native",
					ContainerPort: 9042,
				}, {
					Name:          "inter-node-msg",
					ContainerPort: 8609,
				}, {
					Name:          "intra-node",
					ContainerPort: 7000,
				}, {
					Name:          "tls-intra-node",
					ContainerPort: 7001,
				}, {
					Name:          "mgmt-api-http",
					ContainerPort: 8080,
				}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dc := &CassandraDatacenter{
				TypeMeta:   tt.fields.TypeMeta,
				ObjectMeta: tt.fields.ObjectMeta,
				Spec:       tt.fields.Spec,
				Status:     tt.fields.Status,
			}
			got, err := dc.GetContainerPorts()
			if (err != nil) != tt.wantErr {
				t.Errorf("CassandraDatacenter.GetContainerPorts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CassandraDatacenter.GetContainerPorts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCassandraDatacenter_GetSeedServiceName(t *testing.T) {
	dc := &CassandraDatacenter{
		Spec: CassandraDatacenterSpec{
			ClusterName: "bob",
		},
	}
	want := "bob-seed-service"
	got := dc.GetSeedServiceName()

	if want != got {
		t.Errorf("CassandraDatacenter.GetSeedService() = %v, want %v", got, want)
	}
}

func TestCassandraDatacenter_SplitRacks_balances_racks_when_no_extra_nodes(t *testing.T) {
	rackNodeCounts := SplitRacks(10, 5)
	assert.ElementsMatch(t, rackNodeCounts, []int{2, 2, 2, 2, 2}, "Rack node counts were not balanced")
}

func TestCassandraDatacenter_SplitRacks_balances_racks_when_some_extra_nodes(t *testing.T) {
	rackNodeCounts := SplitRacks(13, 5)
	assert.ElementsMatch(t, rackNodeCounts, []int{3, 3, 3, 2, 2}, "Rack node counts were not balanced")
}
