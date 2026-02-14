package service

type Order struct {
	ID       int     `json:"id"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
	Date     string  `json:"date"`
}

type OrderService interface {
	Create(order Order) Order
	GetAll() []Order
	GetByID(id int) (Order, bool)
	Update(id int, order Order) (Order, bool)
	Delete(id int) bool
}

type orderService struct {
	orders []Order
}

func NewOrderService() OrderService {
	return &orderService{
		orders: []Order{},
	}
}

func (s *orderService) Create(order Order) Order {
	order.ID = len(s.orders) + 1
	s.orders = append(s.orders, order)
	return order
}

func (s *orderService) GetAll() []Order {
	return s.orders
}

func (s *orderService) GetByID(id int) (Order, bool) {
	for _, o := range s.orders {
		if o.ID == id {
			return o, true
		}
	}
	return Order{}, false
}

func (s *orderService) Update(id int, order Order) (Order, bool) {
	for i, o := range s.orders {
		if o.ID == id {
			order.ID = id
			s.orders[i] = order
			return order, true
		}
	}
	return Order{}, false
}

func (s *orderService) Delete(id int) bool {
	for i, o := range s.orders {
		if o.ID == id {
			s.orders = append(s.orders[:i], s.orders[i+1:]...)
			return true
		}
	}
	return false
}
