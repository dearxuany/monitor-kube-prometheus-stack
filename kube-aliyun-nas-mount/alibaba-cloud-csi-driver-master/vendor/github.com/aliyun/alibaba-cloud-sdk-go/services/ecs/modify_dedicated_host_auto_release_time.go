package ecs

//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
//
// Code generated by Alibaba Cloud SDK Code Generator.
// Changes may cause incorrect behavior and will be lost if the code is regenerated.

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
)

// ModifyDedicatedHostAutoReleaseTime invokes the ecs.ModifyDedicatedHostAutoReleaseTime API synchronously
func (client *Client) ModifyDedicatedHostAutoReleaseTime(request *ModifyDedicatedHostAutoReleaseTimeRequest) (response *ModifyDedicatedHostAutoReleaseTimeResponse, err error) {
	response = CreateModifyDedicatedHostAutoReleaseTimeResponse()
	err = client.DoAction(request, response)
	return
}

// ModifyDedicatedHostAutoReleaseTimeWithChan invokes the ecs.ModifyDedicatedHostAutoReleaseTime API asynchronously
func (client *Client) ModifyDedicatedHostAutoReleaseTimeWithChan(request *ModifyDedicatedHostAutoReleaseTimeRequest) (<-chan *ModifyDedicatedHostAutoReleaseTimeResponse, <-chan error) {
	responseChan := make(chan *ModifyDedicatedHostAutoReleaseTimeResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.ModifyDedicatedHostAutoReleaseTime(request)
		if err != nil {
			errChan <- err
		} else {
			responseChan <- response
		}
	})
	if err != nil {
		errChan <- err
		close(responseChan)
		close(errChan)
	}
	return responseChan, errChan
}

// ModifyDedicatedHostAutoReleaseTimeWithCallback invokes the ecs.ModifyDedicatedHostAutoReleaseTime API asynchronously
func (client *Client) ModifyDedicatedHostAutoReleaseTimeWithCallback(request *ModifyDedicatedHostAutoReleaseTimeRequest, callback func(response *ModifyDedicatedHostAutoReleaseTimeResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *ModifyDedicatedHostAutoReleaseTimeResponse
		var err error
		defer close(result)
		response, err = client.ModifyDedicatedHostAutoReleaseTime(request)
		callback(response, err)
		result <- 1
	})
	if err != nil {
		defer close(result)
		callback(nil, err)
		result <- 0
	}
	return result
}

// ModifyDedicatedHostAutoReleaseTimeRequest is the request struct for api ModifyDedicatedHostAutoReleaseTime
type ModifyDedicatedHostAutoReleaseTimeRequest struct {
	*requests.RpcRequest
	ResourceOwnerId      requests.Integer `position:"Query" name:"ResourceOwnerId"`
	ResourceOwnerAccount string           `position:"Query" name:"ResourceOwnerAccount"`
	OwnerAccount         string           `position:"Query" name:"OwnerAccount"`
	AutoReleaseTime      string           `position:"Query" name:"AutoReleaseTime"`
	DedicatedHostId      string           `position:"Query" name:"DedicatedHostId"`
	OwnerId              requests.Integer `position:"Query" name:"OwnerId"`
}

// ModifyDedicatedHostAutoReleaseTimeResponse is the response struct for api ModifyDedicatedHostAutoReleaseTime
type ModifyDedicatedHostAutoReleaseTimeResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
}

// CreateModifyDedicatedHostAutoReleaseTimeRequest creates a request to invoke ModifyDedicatedHostAutoReleaseTime API
func CreateModifyDedicatedHostAutoReleaseTimeRequest() (request *ModifyDedicatedHostAutoReleaseTimeRequest) {
	request = &ModifyDedicatedHostAutoReleaseTimeRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Ecs", "2014-05-26", "ModifyDedicatedHostAutoReleaseTime", "", "")
	request.Method = requests.POST
	return
}

// CreateModifyDedicatedHostAutoReleaseTimeResponse creates a response to parse from ModifyDedicatedHostAutoReleaseTime response
func CreateModifyDedicatedHostAutoReleaseTimeResponse() (response *ModifyDedicatedHostAutoReleaseTimeResponse) {
	response = &ModifyDedicatedHostAutoReleaseTimeResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
