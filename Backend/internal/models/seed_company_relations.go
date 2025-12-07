package models

import (
	"gorm.io/gorm"
)

// SeedCompanyRelations 企業関係のシードデータ（改良版）
func SeedCompanyRelations(db *gorm.DB) error {
	// 既存の企業数を確認
	var count int64
	db.Model(&Company{}).Count(&count)

	if count < 3 {
		// 企業データが不足している場合はスキップ
		return nil
	}

	// 既存の関係数を確認
	var relCount int64
	db.Model(&CompanyRelation{}).Where("(parent_id <= 20 OR child_id <= 20 OR from_id <= 20 OR to_id <= 20)").Count(&relCount)

	if relCount > 100 {
		// 既に基本企業の関係が存在する場合はスキップ
		return nil
	}

	relations := []CompanyRelation{
		// ============ 資本関係図 ============
		// テックホールディングス(1)を親会社とするグループ構造

		// 完全子会社（100%出資）
		{ParentID: ptr(uint(1)), ChildID: ptr(uint(2)), RelationType: "capital_subsidiary", Ratio: ptr(100.0), Description: "完全子会社 - ITソリューション事業の中核"},

		// 関連会社（持分法適用）
		{ParentID: ptr(uint(1)), ChildID: ptr(uint(3)), RelationType: "capital_affiliate", Ratio: ptr(30.0), Description: "関連会社 - コンサルティング事業の戦略的パートナー"},

		// ============ ビジネス関係図（基本3社） ============
		// システム開発・技術サービス
		{FromID: ptr(uint(2)), ToID: ptr(uint(1)), RelationType: "business_service_provider", Description: "【技術サービス】基幹システム開発・運用保守を提供"},
		{FromID: ptr(uint(2)), ToID: ptr(uint(3)), RelationType: "business_partner", Description: "【技術協力】クラウド移行プロジェクトの共同実施"},

		// コンサルティングサービス
		{FromID: ptr(uint(3)), ToID: ptr(uint(1)), RelationType: "business_service_provider", Description: "【経営支援】DX推進・事業戦略コンサルティングを提供"},
		{FromID: ptr(uint(3)), ToID: ptr(uint(2)), RelationType: "business_partner", Description: "【業務提携】ITコンサルティング案件の協業"},

		// グループ内取引
		{FromID: ptr(uint(1)), ToID: ptr(uint(2)), RelationType: "business_internal", Description: "【グループ内】経営管理・人事労務サービスの提供"},
		{FromID: ptr(uint(1)), ToID: ptr(uint(3)), RelationType: "business_investment", Description: "【投資】新規事業開発への資金・リソース提供"},
	}

	// 追加企業（ID 4-20）用のビジネス関係を動的生成
	businessTypes := []struct {
		Type string
		Desc string
	}{
		{"business_service_provider", "【技術サービス】システム開発・運用保守を提供"},
		{"business_service_provider", "【コンサルティング】IT戦略・経営支援を提供"},
		{"business_partner", "【業務提携】共同開発プロジェクトの実施"},
		{"business_partner", "【技術協力】技術ノウハウの相互提供"},
		{"business_supplier", "【製品供給】ソフトウェア・ハードウェアの供給"},
		{"business_outsource", "【業務委託】開発・運用業務の委託"},
	}

	// ID 4-20の企業が互いにビジネス関係を持つように生成
	for i := uint(4); i <= 20; i++ {
		// 各企業が5-10件のビジネス関係を持つ
		numRelations := 5 + (i % 6)
		for j := uint(0); j < numRelations; j++ {
			targetID := ((i + j + 1) % 17) + 4 // 4-20の範囲でループ
			if targetID == i {
				targetID = (targetID % 17) + 4
			}

			typeIdx := (i + j) % uint(len(businessTypes))
			bizType := businessTypes[typeIdx]

			relations = append(relations, CompanyRelation{
				FromID:       ptr(i),
				ToID:         ptr(targetID),
				RelationType: bizType.Type,
				Description:  bizType.Desc,
				IsActive:     true,
			})
		}
	}

	for _, relation := range relations {
		if err := db.FirstOrCreate(&relation, CompanyRelation{
			ParentID:     relation.ParentID,
			ChildID:      relation.ChildID,
			FromID:       relation.FromID,
			ToID:         relation.ToID,
			RelationType: relation.RelationType,
		}).Error; err != nil {
			return err
		}
	}

	return nil
}

// SeedCompanyMarketInfo 企業の市場情報シードデータ（詳細版）
func SeedCompanyMarketInfo(db *gorm.DB) error {
	// 既存の企業数を確認
	var count int64
	db.Model(&Company{}).Count(&count)

	if count < 3 {
		// 企業データが不足している場合はスキップ
		return nil
	}

	marketInfos := []CompanyMarketInfo{
		// テックホールディングス - 東証プライム上場
		{
			CompanyID:   1,
			MarketType:  "prime",
			IsListed:    true,
			StockCode:   "9001",
			MarketCap:   ptr(float64(500000)), // 時価総額: 5000億円
			ListingDate: ptr("2015-04-01"),
		},
		// ITソリューションズ - 非上場（親会社の完全子会社）
		{
			CompanyID:   2,
			MarketType:  "unlisted",
			IsListed:    false,
			StockCode:   "",
			MarketCap:   nil,
			ListingDate: nil,
		},
		// ビジネスコンサルティング - 東証グロース上場
		{
			CompanyID:   3,
			MarketType:  "growth",
			IsListed:    true,
			StockCode:   "9003",
			MarketCap:   ptr(float64(15000)), // 時価総額: 150億円
			ListingDate: ptr("2020-12-15"),
		},
	}

	for _, info := range marketInfos {
		if err := db.FirstOrCreate(&info, CompanyMarketInfo{CompanyID: info.CompanyID}).Error; err != nil {
			return err
		}
	}

	return nil
}

// ポインタヘルパー関数
func ptr[T any](v T) *T {
	return &v
}
