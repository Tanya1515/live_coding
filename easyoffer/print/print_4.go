package main

// странный импорт
import java.net.IDN;

type OrderSevice struct {
    BookingService      BookingService
    UserService         UserService
}
type UserService interface {
    LockUser(User)      error
    UnlockUser(User)    error
}

type User struct{
    ID string
}

type Receipt struct {
    ID          string
    BookingCode string
    BookedAt    string
}

type BookingService interface {
    BookFlight() (string, *BookingSeviceError)
}
type BookingServiceError struct {
    error
    TryAgain bool
}

func (s *OrderService) HandeBookingOrder(user User) *Receipt {
    receipt := Receipt{ID: uuid.New().String()}

    if err := s.UserService.LockUser(user); err != nil {
        log.Logger.Err(err)
        return nil
    }
    for {
        bookingCode, err := s.BookingService.BookFlight()


        switch{
            case err == nil:
                receipt.BookedAt = time.Now().Format(time.RFC3339)
                receipt.BookingCode = bookingCode
                return &receipt
            case err.TryAgain:
			default
                log.Logger.Err(err)
                break
        }
    }
	// код будет недоступен, поскольку break прсото выходит из select-а
    if err := s.UserService.UnlockUser(user); err != nil {
        log.Logger.Err(err)
        return nil
                
    }
    return &receipt
}