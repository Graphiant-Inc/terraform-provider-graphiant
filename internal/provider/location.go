package provider

import (
	graphiant "github.com/Graphiant-Inc/graphiant-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// locationModel mirrors manaV2Location and is embedded as a single nested
// attribute on resources/data sources that carry a site location.
type locationModel struct {
	AddressLine1 types.String  `tfsdk:"address_line1"`
	AddressLine2 types.String  `tfsdk:"address_line2"`
	City         types.String  `tfsdk:"city"`
	State        types.String  `tfsdk:"state"`
	StateCode    types.String  `tfsdk:"state_code"`
	ProvinceCode types.String  `tfsdk:"province_code"`
	Country      types.String  `tfsdk:"country"`
	CountryCode  types.String  `tfsdk:"country_code"`
	Latitude     types.Float64 `tfsdk:"latitude"`
	Longitude    types.Float64 `tfsdk:"longitude"`
	Notes        types.String  `tfsdk:"notes"`
}

func expandLocation(m *locationModel) *graphiant.ManaV2Location {
	if m == nil {
		return nil
	}
	loc := graphiant.NewManaV2LocationWithDefaults()
	if v := strPtr(m.AddressLine1); v != nil {
		loc.SetAddressLine1(*v)
	}
	if v := strPtr(m.AddressLine2); v != nil {
		loc.SetAddressLine2(*v)
	}
	if v := strPtr(m.City); v != nil {
		loc.SetCity(*v)
	}
	if v := strPtr(m.State); v != nil {
		loc.SetState(*v)
	}
	if v := strPtr(m.StateCode); v != nil {
		loc.SetStateCode(*v)
	}
	if v := strPtr(m.ProvinceCode); v != nil {
		loc.SetProvinceCode(*v)
	}
	if v := strPtr(m.Country); v != nil {
		loc.SetCountry(*v)
	}
	if v := strPtr(m.CountryCode); v != nil {
		loc.SetCountryCode(*v)
	}
	if v := float64Ptr(m.Latitude); v != nil {
		loc.SetLatitude(*v)
	}
	if v := float64Ptr(m.Longitude); v != nil {
		loc.SetLongitude(*v)
	}
	if v := strPtr(m.Notes); v != nil {
		loc.SetNotes(*v)
	}
	return loc
}

func flattenLocation(loc *graphiant.ManaV2Location) *locationModel {
	if loc == nil {
		return nil
	}
	return &locationModel{
		AddressLine1: strValue(loc.AddressLine1),
		AddressLine2: strValue(loc.AddressLine2),
		City:         strValue(loc.City),
		State:        strValue(loc.State),
		StateCode:    strValue(loc.StateCode),
		ProvinceCode: strValue(loc.ProvinceCode),
		Country:      strValue(loc.Country),
		CountryCode:  strValue(loc.CountryCode),
		Latitude:     float64Value(loc.Latitude),
		Longitude:    float64Value(loc.Longitude),
		Notes:        strValue(loc.Notes),
	}
}
