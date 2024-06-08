package main

import (
	"errors"
	"net/http"

	pb "github.com/ecgbeald/burgate/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type handler struct {
	orderClient pb.OrderServiceClient
	menuClient  pb.MenuServiceClient
}

func NewHandler(orderCli pb.OrderServiceClient, menuCli pb.MenuServiceClient) *handler {
	return &handler{orderClient: orderCli, menuClient: menuCli}
}

func (h *handler) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/customers/{customerID}/orders", h.HandleCreateOrder)
	mux.HandleFunc("POST /api/menu/entries", h.HandleCreateMenuEntry)
}

func (h *handler) HandleCreateMenuEntry(w http.ResponseWriter, r *http.Request) {
	var entries []*pb.MenuEntry

	if err := ReadJSON(r, &entries); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := validateEntries(entries); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	o, err := h.menuClient.CreateMenuEntry(r.Context(), &pb.MenuEntries{
		Entries: entries,
	})
	s := status.Convert(err)
	if s != nil {
		if s.Code() != codes.InvalidArgument {
			WriteError(w, http.StatusBadRequest, s.Message())
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
	}
	if o.Error {
		WriteError(w, http.StatusUnavailableForLegalReasons, *o.ErrorMsg)
	} else {
		WriteJSON(w, http.StatusOK, o)
	}
}

func (h *handler) HandleCreateOrder(w http.ResponseWriter, r *http.Request) {
	customerID := r.PathValue("customerID")

	var items []*pb.ItemsWithQuantity
	if err := ReadJSON(r, &items); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := validateItems(items); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	o, err := h.orderClient.CreateOrder(r.Context(), &pb.CreateOrderRequest{
		CustomerID: customerID,
		Items:      items,
	})
	// handle grpc error
	s := status.Convert(err)
	if s != nil {
		if s.Code() != codes.InvalidArgument {
			WriteError(w, http.StatusBadRequest, s.Message())
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
	}

	WriteJSON(w, http.StatusOK, o)
}

func validateEntries(entries []*pb.MenuEntry) error {
	if len(entries) == 0 {
		return errors.New("no entries")
	}
	for _, i := range entries {
		if i.ID == "" {
			return errors.New("entry ID is required")
		}

		if i.EntryName == "" {
			return errors.New("entry name is required")
		}

		if i.Price == "" {
			return errors.New("entry price ID is required")
		}
	}
	return nil
}

func validateItems(items []*pb.ItemsWithQuantity) error {
	if len(items) == 0 {
		return errors.New("no items")
	}

	for _, i := range items {
		if i.ID == "" {
			return errors.New("item ID is required")
		}

		if i.Quantity <= 0 {
			return errors.New("item quantity should be positive")
		}
	}
	return nil
}
