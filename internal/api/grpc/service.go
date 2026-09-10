// Copyright 2026 fanjia1024
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package grpc 提供 gRPC 服务端，与 HTTP 能力对齐；调用 Engine 与 DocumentService，不直接调 storage。
package grpc

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Colin4k1024/Aetheris/v2/internal/agent/job"
	"github.com/Colin4k1024/Aetheris/v2/internal/api/grpc/pb"
	appcore "github.com/Colin4k1024/Aetheris/v2/internal/app"
	"github.com/Colin4k1024/Aetheris/v2/internal/pipeline/common"
	"github.com/Colin4k1024/Aetheris/v2/internal/runtime/eino"
	"github.com/Colin4k1024/Aetheris/v2/pkg/auth"
)

// Server gRPC 服务端，持有 Engine、DocumentService 与 JobStore
type Server struct {
	pb.UnimplementedDocumentServiceServer
	pb.UnimplementedQueryServiceServer
	pb.UnimplementedJobServiceServer
	engine     *eino.Engine
	docService appcore.DocumentService
	jobStore   job.JobStore
}

// NewServer 根据注入的 Engine 与 DocumentService 创建 gRPC Server（不含 JobService）
func NewServer(engine *eino.Engine, docService appcore.DocumentService) *Server {
	return &Server{
		engine:     engine,
		docService: docService,
	}
}

// NewServerWithJobStore 创建包含 JobService 的 gRPC Server
func NewServerWithJobStore(engine *eino.Engine, docService appcore.DocumentService, js job.JobStore) *Server {
	s := NewServer(engine, docService)
	s.jobStore = js
	return s
}

// SetJobStore 注入 JobStore（供 JobService 使用）
func (s *Server) SetJobStore(js job.JobStore) {
	s.jobStore = js
}

// Register 注册 Document、Query 与 Job 服务到 grpc.Server
func (s *Server) Register(grpcServer *grpc.Server) {
	pb.RegisterDocumentServiceServer(grpcServer, s)
	pb.RegisterQueryServiceServer(grpcServer, s)
	pb.RegisterJobServiceServer(grpcServer, s)
}

// ListDocuments 实现 DocumentService.ListDocuments
func (s *Server) ListDocuments(ctx context.Context, req *pb.ListDocumentsRequest) (*pb.ListDocumentsResponse, error) {
	tenantID := auth.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = "default"
	}
	docs, err := s.docService.ListDocuments(ctx, tenantID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list documents: %v", err)
	}
	out := make([]*pb.DocumentInfo, len(docs))
	for i, d := range docs {
		out[i] = docInfoToPB(d)
	}
	return &pb.ListDocumentsResponse{
		Documents: out,
		Total:     int32(len(out)),
	}, nil
}

// GetDocument 实现 DocumentService.GetDocument
func (s *Server) GetDocument(ctx context.Context, req *pb.GetDocumentRequest) (*pb.GetDocumentResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}
	doc, err := s.docService.GetDocument(ctx, req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "get document: %v", err)
	}
	return &pb.GetDocumentResponse{Document: docInfoToPB(doc)}, nil
}

// DeleteDocument 实现 DocumentService.DeleteDocument
func (s *Server) DeleteDocument(ctx context.Context, req *pb.DeleteDocumentRequest) (*pb.DeleteDocumentResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}
	if err := s.docService.DeleteDocument(ctx, req.GetId()); err != nil {
		return nil, status.Errorf(codes.Internal, "delete document: %v", err)
	}
	return &pb.DeleteDocumentResponse{Success: true}, nil
}

// UploadDocument 实现 DocumentService.UploadDocument（通过 ingest_pipeline，params 使用 content []byte）
func (s *Server) UploadDocument(ctx context.Context, req *pb.UploadDocumentRequest) (*pb.UploadDocumentResponse, error) {
	if len(req.GetContent()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "content required")
	}
	result, err := s.engine.ExecuteWorkflow(ctx, "ingest_pipeline", map[string]interface{}{
		"content": req.GetContent(),
		"metadata": map[string]interface{}{
			"filename":     req.GetFilename(),
			"content_type": req.GetContentType(),
			"uploaded_at":  time.Now(),
		},
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "upload document: %v", err)
	}
	m, _ := result.(map[string]interface{})
	docID, _ := m["doc_id"].(string)
	return &pb.UploadDocumentResponse{
		Success:    true,
		Message:    "文档上传成功",
		DocumentId: docID,
	}, nil
}

// Query 实现 QueryService.Query
func (s *Server) Query(ctx context.Context, req *pb.QueryRequest) (*pb.QueryResponse, error) {
	if req.GetQuery() == "" {
		return nil, status.Error(codes.InvalidArgument, "query required")
	}
	topK := int(req.GetTopK())
	if topK <= 0 {
		topK = 10
	}
	q := &common.Query{
		ID:        fmt.Sprintf("query-%d", time.Now().UnixNano()),
		Text:      req.GetQuery(),
		Metadata:  nil,
		CreatedAt: time.Now(),
	}
	if req.Metadata != nil {
		q.Metadata = make(map[string]interface{})
		for k, v := range req.Metadata {
			q.Metadata[k] = v
		}
	}
	result, err := s.engine.ExecuteWorkflow(ctx, "query_pipeline", map[string]interface{}{
		"query": q,
		"top_k": topK,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query: %v", err)
	}
	genResult, ok := result.(*common.GenerationResult)
	if !ok {
		return &pb.QueryResponse{Success: true, Answer: fmt.Sprint(result)}, nil
	}
	return &pb.QueryResponse{
		Success:    true,
		Answer:     genResult.Answer,
		References: genResult.References,
	}, nil
}

// BatchQuery 实现 QueryService.BatchQuery
func (s *Server) BatchQuery(ctx context.Context, req *pb.BatchQueryRequest) (*pb.BatchQueryResponse, error) {
	results := make([]*pb.QueryResponse, 0, len(req.GetQueries()))
	for _, q := range req.GetQueries() {
		res, err := s.Query(ctx, q)
		if err != nil {
			results = append(results, &pb.QueryResponse{Success: false, Error: err.Error()})
			continue
		}
		results = append(results, res)
	}
	return &pb.BatchQueryResponse{Results: results}, nil
}

func docInfoToPB(d *appcore.DocumentInfo) *pb.DocumentInfo {
	if d == nil {
		return nil
	}
	return &pb.DocumentInfo{
		Id:          d.ID,
		Name:        d.Name,
		Type:        d.Type,
		Size:        d.Size,
		Path:        d.Path,
		Status:      d.Status,
		Chunks:      int32(d.Chunks),
		VectorCount: int32(d.VectorCount),
		Metadata:    d.Metadata,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}

// ============ JobService 实现 ============

// SubmitJob 实现提交新任务
func (s *Server) SubmitJob(ctx context.Context, req *pb.SubmitJobRequest) (*pb.SubmitJobResponse, error) {
	if s.jobStore == nil {
		return nil, status.Error(codes.Unimplemented, "job store not configured")
	}
	if req.GetAgentId() == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id required")
	}
	tenantID := req.GetTenantId()
	if tenantID == "" {
		tenantID = auth.GetTenantID(ctx)
		if tenantID == "" {
			tenantID = "default"
		}
	}
	j := &job.Job{
		AgentID:   req.GetAgentId(),
		TenantID:  tenantID,
		Goal:      req.GetInput(),
		Status:    job.StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	jobID, err := s.jobStore.Create(ctx, j)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create job: %v", err)
	}
	return &pb.SubmitJobResponse{
		JobId:   jobID,
		Success: true,
		Message: "job submitted",
	}, nil
}

// GetJob 实现获取任务状态
func (s *Server) GetJob(ctx context.Context, req *pb.GetJobRequest) (*pb.GetJobResponse, error) {
	if s.jobStore == nil {
		return nil, status.Error(codes.Unimplemented, "job store not configured")
	}
	if req.GetJobId() == "" {
		return nil, status.Error(codes.InvalidArgument, "job_id required")
	}
	j, err := s.jobStore.Get(ctx, req.GetJobId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get job: %v", err)
	}
	if j == nil {
		return nil, status.Errorf(codes.NotFound, "job %s not found", req.GetJobId())
	}
	return &pb.GetJobResponse{
		Job: jobToPB(j),
	}, nil
}

// CancelJob 实现取消任务
func (s *Server) CancelJob(ctx context.Context, req *pb.CancelJobRequest) (*pb.CancelJobResponse, error) {
	if s.jobStore == nil {
		return nil, status.Error(codes.Unimplemented, "job store not configured")
	}
	if req.GetJobId() == "" {
		return nil, status.Error(codes.InvalidArgument, "job_id required")
	}
	if err := s.jobStore.RequestCancel(ctx, req.GetJobId()); err != nil {
		return &pb.CancelJobResponse{
			Success: false,
			Message: fmt.Sprintf("cancel failed: %v", err),
		}, nil
	}
	return &pb.CancelJobResponse{
		Success: true,
		Message: "cancel requested",
	}, nil
}

// Heartbeat 实现 Worker 心跳
func (s *Server) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	if s.jobStore == nil {
		return nil, status.Error(codes.Unimplemented, "job store not configured")
	}
	// 更新 Job 的 updated_at（通过 UpdateStatus 保持当前状态）
	j, err := s.jobStore.Get(ctx, req.GetJobId())
	if err != nil || j == nil {
		return &pb.HeartbeatResponse{
			Success: false,
			Message: fmt.Sprintf("job %s not found", req.GetJobId()),
		}, nil
	}
	// 保持当前状态，仅更新时间戳
	if err := s.jobStore.UpdateStatus(ctx, req.GetJobId(), j.Status); err != nil {
		return &pb.HeartbeatResponse{
			Success: false,
			Message: fmt.Sprintf("heartbeat failed: %v", err),
		}, nil
	}
	return &pb.HeartbeatResponse{
		Success: true,
		Message: "ok",
	}, nil
}

// ListJobs 实现列出任务
func (s *Server) ListJobs(ctx context.Context, req *pb.ListJobsRequest) (*pb.ListJobsResponse, error) {
	if s.jobStore == nil {
		return nil, status.Error(codes.Unimplemented, "job store not configured")
	}
	if req.GetAgentId() == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id required")
	}
	tenantID := req.GetTenantId()
	if tenantID == "" {
		tenantID = auth.GetTenantID(ctx)
	}
	jobs, err := s.jobStore.ListByAgent(ctx, req.GetAgentId(), tenantID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list jobs: %v", err)
	}
	// 可选状态过滤
	var filtered []*job.Job
	for _, j := range jobs {
		if req.GetStatus() != "" && j.Status.String() != req.GetStatus() {
			continue
		}
		filtered = append(filtered, j)
	}
	// 分页
	limit := int(req.GetLimit())
	offset := int(req.GetOffset())
	if limit <= 0 {
		limit = len(filtered)
	}
	if offset >= len(filtered) {
		filtered = nil
	} else if offset+limit >= len(filtered) {
		filtered = filtered[offset:]
	} else {
		filtered = filtered[offset : offset+limit]
	}
	out := make([]*pb.JobInfo, len(filtered))
	for i, j := range filtered {
		out[i] = jobToPB(j)
	}
	return &pb.ListJobsResponse{
		Jobs:  out,
		Total: int32(len(out)),
	}, nil
}

// jobToPB 将内部 Job 转为 proto JobInfo
func jobToPB(j *job.Job) *pb.JobInfo {
	return &pb.JobInfo{
		JobId:     j.ID,
		AgentId:   j.AgentID,
		Status:    j.Status.String(),
		Input:     j.Goal,
		Output:    "",
		CreatedAt: j.CreatedAt.Unix(),
		UpdatedAt: j.UpdatedAt.Unix(),
		Attempt:   int32(j.RetryCount),
		Error:     "",
	}
}
