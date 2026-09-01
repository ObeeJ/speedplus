package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
)

func TestMerchantCRUD_AllFiveVerticals(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		ctx := context.Background()

		verticals := []struct {
			vertical    model.MerchantVertical
			name        string
			productName string
			priceKobo   int64
			category    string
			licence     *string
			isGasPlant  bool
		}{
			{
				vertical:    model.VerticalGas,
				name:        "Lagos Gas Plant Ltd",
				productName: "12.5kg Refill & Swap",
				priceKobo:   1250000,
				category:    "LPG Gas",
				isGasPlant:  true,
			},
			{
				vertical:    model.VerticalFood,
				name:        "Mama Cass Kitchen",
				productName: "Jollof Rice & Fried Chicken",
				priceKobo:   450000,
				category:    "Main Dishes",
			},
			{
				vertical:    model.VerticalPharmacy,
				name:        "HealthPlus Pharmacy",
				productName: "Paracetamol 500mg (Pack of 10)",
				priceKobo:   80000,
				category:    "Pain Relief",
				licence:     stringPtr("PCN-994812"),
			},
			{
				vertical:    model.VerticalGrocery,
				name:        "Shoprite Lekki",
				productName: "Golden Penny Semovita 2kg",
				priceKobo:   280000,
				category:    "Grains & Flour",
			},
			{
				vertical:    model.VerticalPackage,
				name:        "Fourdat Express Logistics",
				productName: "Intra-City Express Parcel",
				priceKobo:   150000,
				category:    "Courier Services",
			},
		}

		for _, v := range verticals {
			t.Run(fmt.Sprintf("Vertical_%s", v.vertical), func(t *testing.T) {
				// 1. CREATE USER & MERCHANT PROFILE
				ownerID := uuid.New()
				owner := &model.User{
					ID:           ownerID,
					Role:         model.RoleMerchant,
					FirstName:    "Merchant",
					LastName:     string(v.vertical),
					Phone:        "+23481" + uuid.NewString()[:8],
					PasswordHash: "hashedpass",
					ReferralCode: "REF" + uuid.NewString()[:6],
				}
				mustCreate(t, tx, owner)

				capacity := 5000
				floatCount := 120
				merchant := &model.Merchant{
					ID:              uuid.New(),
					UserID:          ownerID,
					BusinessName:    v.name,
					Vertical:        v.vertical,
					Status:          model.MerchantActive,
					IsOpen:          true,
					Lat:             6.5244,
					Lng:             3.3792,
					IsGasPlant:      v.isGasPlant,
					PlantCapacityKg: &capacity,
					FloatCount:      &floatCount,
				}
				mustCreate(t, tx, merchant)

				// Also seed MerchantProfile for handler compatibility
				profile := &model.MerchantProfile{
					ID:            uuid.New(),
					UserID:        ownerID,
					BusinessName:  v.name,
					Vertical:      v.vertical,
					Status:        model.MerchantActive,
					IsOpen:        true,
					LicenceNumber: v.licence,
				}
				mustCreate(t, tx, profile)

				// 2. CREATE PRODUCT (CREATE)
				desc := "High quality product for " + string(v.vertical)
				product := &model.Product{
					ID:          uuid.New(),
					MerchantID:  merchant.ID,
					Name:        v.productName,
					Description: &desc,
					PriceKobo:   v.priceKobo,
					Category:    v.category,
					IsAvailable: true,
				}
				mustCreate(t, tx, product)

				// 3. READ PRODUCT (READ)
				var fetched model.Product
				if err := tx.WithContext(ctx).First(&fetched, "id = ?", product.ID).Error; err != nil {
					t.Fatalf("Read product failed: %v", err)
				}
				if fetched.Name != v.productName {
					t.Errorf("Product name = %q, want %q", fetched.Name, v.productName)
				}
				if fetched.PriceKobo != v.priceKobo {
					t.Errorf("Product price = %d, want %d", fetched.PriceKobo, v.priceKobo)
				}

				// 4. UPDATE PRODUCT (UPDATE)
				newPrice := v.priceKobo + 50000 // Increase price by 500 NGN
				if err := tx.WithContext(ctx).Model(&model.Product{}).
					Where("id = ?", product.ID).
					Updates(map[string]interface{}{
						"price_kobo":   newPrice,
						"is_available": false,
					}).Error; err != nil {
					t.Fatalf("Update product failed: %v", err)
				}

				var updated model.Product
				tx.WithContext(ctx).First(&updated, "id = ?", product.ID)
				if updated.PriceKobo != newPrice {
					t.Errorf("Updated price = %d, want %d", updated.PriceKobo, newPrice)
				}
				if updated.IsAvailable {
					t.Errorf("IsAvailable = true, want false")
				}

				// 5. READ MERCHANT STATUS & TOGGLE OPEN/CLOSED
				if err := tx.WithContext(ctx).Model(&model.Merchant{}).
					Where("id = ?", merchant.ID).
					Update("is_open", false).Error; err != nil {
					t.Fatalf("Toggle merchant open state failed: %v", err)
				}

				var toggled model.Merchant
				tx.WithContext(ctx).First(&toggled, "id = ?", merchant.ID)
				if toggled.IsOpen {
					t.Errorf("Merchant IsOpen = true, want false")
				}

				// 6. DELETE PRODUCT (DELETE)
				if err := tx.WithContext(ctx).Delete(&model.Product{}, "id = ?", product.ID).Error; err != nil {
					t.Fatalf("Delete product failed: %v", err)
				}

				var deletedCheck model.Product
				err := tx.WithContext(ctx).First(&deletedCheck, "id = ?", product.ID).Error
				if err == nil {
					t.Errorf("Expected product to be deleted, but found it")
				}
			})
		}
	})
}

func stringPtr(s string) *string {
	return &s
}
