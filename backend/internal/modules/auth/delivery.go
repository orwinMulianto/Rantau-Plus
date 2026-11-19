package auth

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/url"
	"time"
)

type EmailSender interface {
	Send(context.Context, string, string, string) error
}

type EmailLinks struct {
	Verify *url.URL
	Reset  *url.URL
}

func (s *Service) RunEmailDelivery(ctx context.Context, sender EmailSender, links EmailLinks) {
	deliveryTicker := time.NewTicker(2 * time.Second)
	defer deliveryTicker.Stop()
	cleanupTicker := time.NewTicker(time.Hour)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-deliveryTicker.C:
			if err := s.deliverEmail(ctx, sender, links); err != nil && ctx.Err() == nil {
				log.Printf("Rantau+ auth: email delivery failed (%T)", err)
			}
		case <-cleanupTicker.C:
			cleanupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := s.repo.cleanExpired(cleanupCtx)
			cancel()
			if err != nil && ctx.Err() == nil {
				log.Printf("Rantau+ auth: cleanup failed (%T)", err)
			}
		}
	}
}

func (s *Service) deliverEmail(ctx context.Context, sender EmailSender, links EmailLinks) error {
	itemCtx, cancel := context.WithTimeout(ctx, 9*time.Second)
	defer cancel()
	item, err := s.repo.claimDelivery(itemCtx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	subject := "Password Rantau+ berubah"
	message := "Password akun Rantau+ kamu baru saja diubah. Semua sesi lama telah dicabut. Jika perubahan ini tidak kamu lakukan, segera gunakan fitur Lupa password.\n"
	if item.purpose != passwordChanged {
		token := emailToken(s.emailKey, item.nonce)
		if digest(token) != item.hash {
			return errors.New("email token key mismatch")
		}
		subject = "Verifikasi email Rantau+"
		message = "Gunakan token ini untuk memverifikasi email dan menyelesaikan pendaftaran. Token berlaku 24 jam.\n"
		link := links.Verify
		if item.purpose == resetPassword {
			subject = "Reset password Rantau+"
			message = "Gunakan token ini untuk mengganti password. Token berlaku 30 menit.\n"
			link = links.Reset
		}
		message += "\n" + emailAction(link, token) + "\n\nAbaikan email ini jika kamu tidak meminta kode ini.\n"
	}
	if err := sender.Send(itemCtx, item.email, subject, message); err != nil {
		return err
	}
	return s.repo.markDelivered(itemCtx, item.hash)
}

func emailAction(base *url.URL, token string) string {
	if base == nil {
		return token
	}
	link := *base
	link.Fragment = "token=" + token
	return link.String()
}
