local distributionTable = VehicleDistributions[1]

VehicleDistributions.MINI69GloveBox = {
    rolls = 1,
    items = {
        "Base.69miniMagazine", 60,
        "Base.Pen", 4,
        "Base.Pencil", 4,
        "Base.Cigarettes", 5,
        "Base.Lighter", 5,
        "Base.Matches", 3,
        "Base.Tissue", 2,
    },
    junk = ClutterTables.GloveBoxJunk,
}

VehicleDistributions.MiniUnionJack = {
    rolls = 5,
    items = {
        "Base.69miniMeatballBulldog", 10,
        "Base.69miniCrateOfTea", 15,
        "Base.69miniRupertBear", 9,
        "Base.PetrolCan", 10,
        "Base.EmptySandbag", 4,
        "Base.Garbagebag", 6,
        "Base.Plasticbag", 10,
        "Base.PopBottleEmpty", 4,
        "Base.PopEmpty", 4,
        "Base.RubberBand", 6,
        "Base.Tarp", 10,
        "Base.Tissue", 10,
        "Base.Tote", 6,
        "Base.Twine", 10,
    }
}

VehicleDistributions.MiniItalianJob = {
    rolls = 1,
    items = {
        "Base.69miniGoldBullions", 35,
        "Base.DuctTape", 8,
        "Base.EmptyPetrolCan", 10,
        "Base.EmptySandbag", 4,
        "Base.Garbagebag", 6,
        "Base.Plasticbag", 10,
        "Base.PopBottleEmpty", 4,
        "Base.PopEmpty", 4,
        "Base.RubberBand", 6,
        "Base.Tarp", 10,
        "Base.Tissue", 10,
        "Base.Tote", 6,
        "Base.Twine", 10,
        "Base.WaterBottleEmpty", 4,
        "Base.WhiskeyEmpty", 1,
    }
}

VehicleDistributions.MiniMrBean = {
    rolls = 1,
    items = {
        "Base.69miniTeddy", 100,
    }
}

VehicleDistributions.MINI69 = {

	GloveBox = VehicleDistributions.MINI69GloveBox;
	MINI69Trunk = VehicleDistributions.TrunkStandard;
}

VehicleDistributions.MINI69UJ = {

	GloveBox = VehicleDistributions.MINI69GloveBox;
	MINI69Trunk = VehicleDistributions.MiniUnionJack;
}

VehicleDistributions.MINI69IJ = {

	GloveBox = VehicleDistributions.MINI69GloveBox;
	MINI69Trunk = VehicleDistributions.MiniItalianJob;
}

VehicleDistributions.MINI69MrB = {

	GloveBox = VehicleDistributions.MINI69GloveBox;
	MINI69Trunk = VehicleDistributions.TrunkStandard;
	SeatRearLeft = VehicleDistributions.MiniMrBean;
}

distributionTable["69mini"] = { Normal = VehicleDistributions.MINI69; }
distributionTable["69miniUnionJack"] = { Normal = VehicleDistributions.MINI69UJ; }
distributionTable["69miniIJ"] = { Normal = VehicleDistributions.MINI69IJ; }
distributionTable["69miniMrB"] = { Normal = VehicleDistributions.MINI69MrB; }
distributionTable["69miniPS"] = { Normal = VehicleDistributions.MINI69; }