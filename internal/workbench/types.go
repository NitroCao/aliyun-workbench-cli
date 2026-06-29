package workbench

type apiResponse[T any] struct {
	Success        bool   `json:"success"`
	HTTPStatusCode any    `json:"httpStatusCode"`
	Root           T      `json:"root"`
	Message        string `json:"message"`
	RequestSerial  string `json:"requestSerialId"`
	Recipient      string `json:"recipient"`
}

type resourceListRoot struct {
	Total     int        `json:"total"`
	Resources []Resource `json:"resources"`
}

type Resource struct {
	ResourceGroupID          string `json:"resourceGroupId"`
	ResourceGroupName        string `json:"resourceGroupName"`
	ResourceGroupDisplayName string `json:"resourceGroupDisplayName"`
	RegionID                 string `json:"regionId"`
	ResourceID               string `json:"resourceId"`
	ResourceName             string `json:"resourceName"`
	PublicIPAddress          string `json:"publicIpAddress"`
	EIPAddress               string `json:"eipAddress"`
	InnerIPAddress           string `json:"innerIpAddress"`
	PrivateIPAddress         string `json:"privateIpAddress"`
	OSType                   string `json:"osType"`
	OSName                   string `json:"osName"`
	RunningStatus            string `json:"runningStatus"`
	IPAddressType            string `json:"ipAddressType"`
	RecommendIPAddress       string `json:"recommendIpAddress"`
}

func (r Resource) loginHost() (host string, ipAddressType string) {
	if r.PrivateIPAddress != "" {
		return r.PrivateIPAddress, "PrivateIpAddress"
	}
	if r.PublicIPAddress != "" {
		return r.PublicIPAddress, "PublicIpAddress"
	}
	if r.EIPAddress != "" {
		return r.EIPAddress, "EipAddress"
	}
	if r.RecommendIPAddress != "" {
		return r.RecommendIPAddress, r.IPAddressType
	}
	return "", ""
}

type LoginResult struct {
	Info LoginInfo
	Root map[string]any
}

type LoginInfo struct {
	CreateTime         string `json:"createTime"`
	ExpireTime         string `json:"expireTime"`
	InstanceType       string `json:"instanceType"`
	RegionID           string `json:"regionId"`
	InstanceID         string `json:"instanceId"`
	InstanceName       string `json:"instanceName"`
	Host               string `json:"host"`
	Port               int    `json:"port"`
	Protocol           string `json:"protocol"`
	NetworkAccessMode  string `json:"networkAccessMode"`
	InstanceLoginToken string `json:"instanceLoginToken"`
	Username           string `json:"username"`
	AuthenticationType string `json:"authenticationType"`
	LoginSuccess       bool   `json:"loginSuccess"`
	ErrorMessage       string `json:"errorMessage"`
	RunningStatus      string `json:"runningStatus"`
}

type ecsListResponse struct {
	Code            string `json:"code"`
	Message         string `json:"message"`
	SuccessResponse *bool  `json:"successResponse"`
	Data            struct {
		Instances struct {
			Instance []ECSInstance `json:"Instance"`
		} `json:"Instances"`
	} `json:"data"`
}

type ECSInstance struct {
	InstanceID   string `json:"InstanceId"`
	InstanceName string `json:"InstanceName"`
	Status       string `json:"Status"`
	OSName       string `json:"OSName"`
	OSType       string `json:"OSType"`
	VPC          struct {
		PrivateIP struct {
			IPAddress []string `json:"IpAddress"`
		} `json:"PrivateIpAddress"`
	} `json:"VpcAttributes"`
	PublicIP struct {
		IPAddress []string `json:"IpAddress"`
	} `json:"PublicIpAddress"`
	EIP struct {
		IPAddress string `json:"IpAddress"`
	} `json:"EipAddress"`
}

func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (i ECSInstance) PrivateIPAddress() string { return first(i.VPC.PrivateIP.IPAddress) }
func (i ECSInstance) PublicIPAddress() string  { return first(i.PublicIP.IPAddress) }
