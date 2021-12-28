package blog

import (
	"context"
	"github.com/dtm-labs/dtm/dtmgrpc"
	"github.com/google/uuid"
	authv1 "github.com/jxlwqq/blog-microservices/api/protobuf/auth/v1"
	v1 "github.com/jxlwqq/blog-microservices/api/protobuf/blog/v1"
	commentv1 "github.com/jxlwqq/blog-microservices/api/protobuf/comment/v1"
	postv1 "github.com/jxlwqq/blog-microservices/api/protobuf/post/v1"
	userv1 "github.com/jxlwqq/blog-microservices/api/protobuf/user/v1"
	"github.com/jxlwqq/blog-microservices/internal/pkg/config"
	"github.com/jxlwqq/blog-microservices/internal/pkg/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var prefix = "/" + v1.BlogService_ServiceDesc.ServiceName + "/"

var AuthMethods = map[string]bool{
	prefix + "SignUp":        false, // 不需要验证
	prefix + "SignIn":        false,
	prefix + "CreatePost":    true, // 需要验证
	prefix + "UpdatePost":    true,
	prefix + "CreateComment": true,
	prefix + "GetPost":       false,
	prefix + "ListPosts":     false,
}

func NewServer(logger *log.Logger,
	conf *config.Config,
	userClient userv1.UserServiceClient,
	postClient postv1.PostServiceClient,
	commentClient commentv1.CommentServiceClient,
	authClient authv1.AuthServiceClient,
) v1.BlogServiceServer {
	return &Server{
		logger:        logger,
		conf:          conf,
		userClient:    userClient,
		postClient:    postClient,
		commentClient: commentClient,
		authClient:    authClient,
	}
}

type Server struct {
	v1.UnimplementedBlogServiceServer
	logger        *log.Logger
	conf          *config.Config
	userClient    userv1.UserServiceClient
	postClient    postv1.PostServiceClient
	commentClient commentv1.CommentServiceClient
	authClient    authv1.AuthServiceClient
}

func (s Server) CreatePost(ctx context.Context, req *v1.CreatePostRequest) (*v1.CreatePostResponse, error) {
	err := req.ValidateAll()
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	userID, ok := ctx.Value("ID").(uint64)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}
	userResp, err := s.userClient.GetUser(ctx, &userv1.GetUserRequest{
		Id: userID,
	})
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	title := req.GetPost().GetTitle()
	content := req.GetPost().GetContent()

	postResp, err := s.postClient.CreatePost(ctx, &postv1.CreatePostRequest{
		Post: &postv1.Post{
			Uuid:    uuid.New().String(),
			Title:   title,
			Content: content,
			UserId:  userResp.GetUser().GetId(),
		},
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1.CreatePostResponse{
		Post: &v1.Post{
			Id:            postResp.GetPost().GetId(),
			Title:         postResp.GetPost().GetTitle(),
			Content:       postResp.GetPost().GetContent(),
			UserId:        postResp.GetPost().GetUserId(),
			CommentsCount: postResp.GetPost().GetCommentsCount(),
			CreatedAt:     postResp.GetPost().GetCreatedAt(),
			UpdatedAt:     postResp.GetPost().GetUpdatedAt(),
			User: &v1.User{
				Id:       userResp.GetUser().GetId(),
				Username: userResp.GetUser().GetUsername(),
				Avatar:   userResp.GetUser().GetAvatar(),
			},
		},
	}, nil

}

func (s Server) GetPost(ctx context.Context, req *v1.GetPostRequest) (*v1.GetPostResponse, error) {
	postID := req.GetId()
	postResp, err := s.postClient.GetPost(ctx, &postv1.GetPostRequest{
		Id: postID,
	})
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	postUserResp, err := s.userClient.GetUser(ctx, &userv1.GetUserRequest{
		Id: postResp.GetPost().GetUserId(),
	})

	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &v1.GetPostResponse{Post: &v1.Post{
		Id:            postResp.GetPost().GetId(),
		Title:         postResp.GetPost().GetTitle(),
		Content:       postResp.GetPost().GetContent(),
		UserId:        postResp.GetPost().GetUserId(),
		CommentsCount: postResp.GetPost().GetCommentsCount(),
		CreatedAt:     postResp.GetPost().GetCreatedAt(),
		UpdatedAt:     postResp.GetPost().GetUpdatedAt(),
		User: &v1.User{
			Id:       postUserResp.GetUser().GetId(),
			Username: postUserResp.GetUser().GetUsername(),
			Avatar:   postUserResp.GetUser().GetAvatar(),
		},
	}}, nil
}

func (s Server) UpdatePost(ctx context.Context, req *v1.UpdatePostRequest) (*v1.UpdatePostResponse, error) {
	userID, ok := ctx.Value("ID").(uint64)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}
	userResp, err := s.userClient.GetUser(ctx, &userv1.GetUserRequest{
		Id: userID,
	})
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	postResp, err := s.postClient.GetPost(ctx, &postv1.GetPostRequest{
		Id: req.GetPost().GetId(),
	})
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	// 授权检查，只能修改自己发布的文章
	if userResp.GetUser().GetId() != postResp.GetPost().GetUserId() {
		return nil, status.Error(codes.PermissionDenied, "user not authorized")
	}

	updatedPost := &postv1.Post{
		Id: req.GetPost().GetId(),
	}

	if req.GetPost().GetTitle() != "" {
		updatedPost.Title = req.GetPost().GetTitle()
	}

	if req.GetPost().GetContent() != "" {
		updatedPost.Content = req.GetPost().GetContent()
	}

	updatePostResp, err := s.postClient.UpdatePost(ctx, &postv1.UpdatePostRequest{
		Post: updatedPost,
	})

	if err != nil || !updatePostResp.GetSuccess() {
		return nil, status.Error(codes.Internal, "hhhh")
	}

	return &v1.UpdatePostResponse{
		Success: true,
	}, nil

}

func (s Server) ListPosts(ctx context.Context, req *v1.ListPostsRequest) (*v1.ListPostsResponse, error) {
	offset := req.GetOffset()
	limit := req.GetLimit()
	postResp, err := s.postClient.ListPosts(ctx, &postv1.ListPostsRequest{
		Offset: int32(offset),
		Limit:  int32(limit),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var posts []*v1.Post

	var postUserIDs []uint64

	for _, post := range postResp.GetPosts() {
		postUserIDs = append(postUserIDs, post.GetUserId())
	}

	postUserResp, err := s.userClient.ListUsersByIDs(ctx, &userv1.ListUsersByIDsRequest{
		Ids: postUserIDs,
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	for _, post := range postResp.GetPosts() {
		for _, postUser := range postUserResp.GetUsers() {
			if post.GetUserId() == postUser.GetId() {
				posts = append(posts, &v1.Post{
					Id:            post.GetId(),
					Title:         post.GetTitle(),
					Content:       post.GetContent(),
					UserId:        post.GetUserId(),
					CommentsCount: post.GetCommentsCount(),
					CreatedAt:     post.GetCreatedAt(),
					UpdatedAt:     post.GetUpdatedAt(),
					User: &v1.User{
						Id:       postUser.GetId(),
						Username: postUser.GetUsername(),
						Avatar:   postUser.GetAvatar(),
					},
				})
			}
		}
	}

	return &v1.ListPostsResponse{
		Posts: posts,
		Total: 0,
	}, nil
}

func (s Server) CreateComment(ctx context.Context, req *v1.CreateCommentRequest) (*v1.CreateCommentResponse, error) {
	userID, ok := ctx.Value("ID").(uint64)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}
	userResp, err := s.userClient.GetUser(ctx, &userv1.GetUserRequest{
		Id: userID,
	})
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	postID := req.GetComment().GetPostId()
	postResp, err := s.postClient.GetPost(ctx, &postv1.GetPostRequest{
		Id: postID,
	})
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	comment := &commentv1.Comment{
		Uuid:    uuid.New().String(),
		PostId:  postResp.GetPost().GetId(),
		UserId:  userResp.GetUser().GetId(),
		Content: req.GetComment().GetContent(),
	}

	// 分布式事务(Saga 模式)
	DtmGrpcServerAddr := s.conf.DTM.Server.Host + s.conf.DTM.Server.GRPC.Port
	gid := dtmgrpc.MustGenGid(DtmGrpcServerAddr)
	s.logger.Info("gid:", gid)
	saga := dtmgrpc.NewSagaGrpc(DtmGrpcServerAddr, gid).Add(
		s.conf.Comment.Server.Host+s.conf.Comment.Server.GRPC.Port+"/"+commentv1.CommentService_ServiceDesc.ServiceName+"/CreateComment",
		s.conf.Comment.Server.Host+s.conf.Comment.Server.GRPC.Port+"/"+commentv1.CommentService_ServiceDesc.ServiceName+"/CreateCommentCompensate",
		&commentv1.CreateCommentRequest{
			Comment: comment,
		},
	).Add(
		s.conf.Post.Server.Host+s.conf.Post.Server.GRPC.Port+"/"+postv1.PostService_ServiceDesc.ServiceName+"/IncrementCommentCount",
		s.conf.Post.Server.Host+s.conf.Post.Server.GRPC.Port+"/"+postv1.PostService_ServiceDesc.ServiceName+"/IncrementCommentCountCompensate",
		&postv1.IncrementCommentCountRequest{
			Id: postID,
		},
	)
	saga.WaitResult = true
	err = saga.Submit()
	if err != nil {
		s.logger.Error("saga submit error:", err)
		return nil, status.Error(codes.Internal, "saga submit failed")
	}

	postUserResp, err := s.userClient.GetUser(ctx, &userv1.GetUserRequest{
		Id: postResp.GetPost().GetUserId(),
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	commentResp, err := s.commentClient.GetCommentByUUID(ctx, &commentv1.GetCommentByUUIDRequest{
		Uuid: comment.GetUuid(),
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1.CreateCommentResponse{
		Comment: &v1.Comment{
			Id:        commentResp.GetComment().GetId(),
			Content:   commentResp.GetComment().GetContent(),
			PostId:    commentResp.GetComment().GetPostId(),
			UserId:    commentResp.GetComment().GetUserId(),
			CreatedAt: commentResp.GetComment().GetCreatedAt(),
			UpdatedAt: commentResp.GetComment().GetUpdatedAt(),
			Post: &v1.Post{
				Id:            postResp.GetPost().GetId(),
				Title:         postResp.GetPost().GetTitle(),
				Content:       postResp.GetPost().GetContent(),
				UserId:        postResp.GetPost().GetUserId(),
				CommentsCount: postResp.GetPost().GetCommentsCount(),
				CreatedAt:     postResp.GetPost().GetCreatedAt(),
				UpdatedAt:     postResp.GetPost().GetUpdatedAt(),
				User: &v1.User{
					Id:       postUserResp.GetUser().GetId(),
					Username: postUserResp.GetUser().GetUsername(),
					Avatar:   postUserResp.GetUser().GetAvatar(),
				},
			},
			User: &v1.User{
				Id:       userResp.GetUser().GetId(),
				Username: userResp.GetUser().GetUsername(),
				Avatar:   userResp.GetUser().GetAvatar(),
			},
		},
	}, nil
}

func (s Server) ListCommentsByPostID(ctx context.Context, req *v1.ListCommentsByPostIDRequest) (*v1.ListCommentsByPostIDResponse, error) {
	postID := req.GetPostId()
	offset := req.GetOffset()
	limit := req.GetLimit()
	commentResp, err := s.commentClient.ListCommentsByPostID(ctx, &commentv1.ListCommentsByPostIDRequest{
		PostId: postID,
		Offset: int32(offset),
		Limit:  int32(limit),
	})
	if err != nil {
		return nil, err
	}

	var comments []*v1.Comment
	for _, comment := range commentResp.GetComments() {
		comments = append(comments, &v1.Comment{
			Id:        comment.GetId(),
			Content:   comment.GetContent(),
			PostId:    comment.GetPostId(),
			UserId:    comment.GetUserId(),
			CreatedAt: comment.GetCreatedAt(),
			UpdatedAt: comment.GetUpdatedAt(),
		})
	}

	return &v1.ListCommentsByPostIDResponse{
		Comments: comments,
		Total:    0,
	}, nil
}

func (s Server) SignUp(ctx context.Context, req *v1.SignUpRequest) (*v1.SignUpResponse, error) {
	err := req.ValidateAll()
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	username := req.GetUsername()
	email := req.GetEmail()
	password := req.GetPassword()

	usernameResp, err := s.userClient.GetUserByUsername(ctx, &userv1.GetUserByUsernameRequest{
		Username: username,
	})
	s.logger.Info("usernameResp:", usernameResp)
	if err == nil && usernameResp.GetUser().GetId() != 0 {
		return nil, status.Error(codes.AlreadyExists, "username already exists")
	}

	emailResp, err := s.userClient.GetUserByEmail(ctx, &userv1.GetUserByEmailRequest{
		Email: email,
	})
	if err == nil && emailResp.GetUser().GetId() != 0 {
		return nil, status.Error(codes.AlreadyExists, "email already exists")
	}

	userResp, err := s.userClient.CreateUser(ctx, &userv1.CreateUserRequest{
		User: &userv1.User{
			Username: username,
			Email:    email,
			Password: password,
		},
	})
	if err != nil {
		s.logger.Error(err)
		return nil, status.Error(codes.Internal, "failed to create user")
	}

	authResp, err := s.authClient.GenerateToken(ctx, &authv1.GenerateTokenRequest{
		UserId: userResp.GetUser().GetId(),
	})

	if err != nil {
		s.logger.Error(err)
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	return &v1.SignUpResponse{
		Token: authResp.GetToken(),
	}, nil
}

func (s Server) SignIn(ctx context.Context, req *v1.SignInRequest) (*v1.SignInResponse, error) {
	err := req.ValidateAll()
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	email := req.GetEmail()
	username := req.GetUsername()
	password := req.GetPassword()
	if email == "" && username == "" {
		return nil, status.Errorf(codes.InvalidArgument, "email and username cannot be empty")
	}
	var userID uint64

	if email != "" {
		resp, err := s.userClient.GetUserByEmail(ctx, &userv1.GetUserByEmailRequest{
			Email:    email,
			Password: password,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get user by email: %v", err)
		}
		user := resp.GetUser()
		userID = user.GetId()
	} else {
		req, err := s.userClient.GetUserByUsername(ctx, &userv1.GetUserByUsernameRequest{
			Username: username,
			Password: password,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get user by username: %v", err)
		}
		user := req.GetUser()
		userID = user.GetId()
	}

	authResp, err := s.authClient.GenerateToken(ctx, &authv1.GenerateTokenRequest{
		UserId: userID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate token: %v", err)
	}

	return &v1.SignInResponse{
		Token: authResp.GetToken(),
	}, nil
}
