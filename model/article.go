package model

type ArticleWrapper struct {
	Article Article `json:"article"`
}

type Article struct {
	Title       string   `form:"title" json:"title" binding:"required,min=4"`
	Description string   `form:"description" json:"description" binding:"omitempty,max=2048"`
	Body        string   `form:"body" json:"body" binding:"omitempty,max=2048"`
	Tags        []string `form:"tagList" json:"tagList"`
}


type UpdateArticleWrapper struct {
	Article UpdateArticle `json:"article"`
}

type UpdateArticle struct {
	Title       string   `form:"title" json:"title" binding:"omitempty,min=4"`
	Description string   `form:"description" json:"description" binding:"omitempty,max=2048"`
	Body        string   `form:"body" json:"body" binding:"omitempty,max=2048"`
	Tags        []string `form:"tagList" json:"tagList"`
}

type ArticleResponse struct {
	ID             uint            `json:"-"`
	Title          string          `json:"title,omitempty"`
	Slug           string          `json:"slug,omitempty"`
	Description    string          `json:"description,omitempty"`
	Body           string          `json:"body,omitempty"`
	CreatedAt      string          `json:"createdAt,omitempty"`
	UpdatedAt      string          `json:"updatedAt,omitempty"`
	Author         ProfileResponse `json:"author,omitempty"`
	Tags           []string        `json:"tagList,omitempty"`
	Favorite       bool            `json:"favorited,omitempty"`
	FavoritesCount uint            `json:"favoritesCount,omitempty"`
}

type CommentWrapper struct {
	Comment Comment `json:"comment"`
}

type Comment struct {
	Body string `form:"body" json:"body" binding:"required,max=2048"`
}

type CommentResponse struct {
	ID        uint            `json:"-"`
	Body      string          `json:"body,omitempty"`
	CreatedAt string          `json:"createdAt,omitempty"`
	UpdatedAt string          `json:"updatedAt,omitempty"`
	Author    ProfileResponse `json:"author,omitempty"`
}
