package usecase

import (
	"cmp"
	"slices"

	"booking/internal/domain/request"
	"booking/internal/dto"
)

func toRequest(req *request.Request) dto.Request {
	items := slices.Clone(req.Items)
	slices.SortFunc(items, func(a, b request.Item) int {
		return cmp.Or(compareDates(a.Date, b.Date), cmp.Compare(a.ServiceID, b.ServiceID))
	})

	output := dto.Request{
		ID:          req.ID,
		UserID:      req.UserID,
		Items:       make([]dto.RequestItem, 0, len(items)),
		Status:      string(req.Status),
		Comment:     req.Comment,
		CreatedAt:   req.CreatedAt,
		ProcessedAt: req.ProcessedAt,
		ProcessedBy: req.ProcessedBy,
	}
	for _, item := range items {
		output.Items = append(output.Items, dto.RequestItem{ServiceID: item.ServiceID, Date: item.Date.String()})
	}

	return output
}

func toRequests(reqs []*request.Request) dto.RequestsOutput {
	output := dto.RequestsOutput{Requests: make([]dto.Request, 0, len(reqs))}
	for _, req := range reqs {
		output.Requests = append(output.Requests, toRequest(req))
	}

	return output
}
