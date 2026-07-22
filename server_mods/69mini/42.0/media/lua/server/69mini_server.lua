require "DAMN_Parts";
require "DAMN_Spawns";

--***********************************************************
--**                   KI5 / bikinihorst                   **
--***********************************************************

DAMN.Parts:processConfigV2("MINI69", {
	["BumperFront"] = {
		partId = "DAMNBumperFront",
		itemToModel = {
			["Base.69miniBumperFront0"] = "FrontBumper0",
			["Base.69miniBumperFront1"] = "FrontBumper1",
			["Base.69miniBumperFront2"] = "FrontBumper2",
		},
		default = "trve_random",
		noPartChance = 5,
	},
	["BumperFrontIJ"] = {
		partId = "DAMNBumperFront",
		itemToModel = {
			["Base.69miniBumperFront2"] = "FrontBumper2",
			["Base.69miniBumperFront0"] = "FrontBumper0",
			["Base.69miniBumperFront1"] = "FrontBumper1",
		},
		default = "first",
	},
	["BumperFrontPS"] = {
		partId = "DAMNBumperFront",
		itemToModel = {
			["Base.69miniBumperFront0"] = "FrontBumper0",
			["Base.69miniBumperFront1"] = "FrontBumper1",
			["Base.69miniBumperFront2"] = "FrontBumper2",
		},
	},
	["BumperRear"] = {
		partId = "DAMNBumperRear",
		itemToModel = {
			["Base.69miniBumperRear0"] = "BumperRear0",
			["Base.69miniBumperRear1"] = "BumperRear1",
		},
		default = "trve_random",
		noPartChance = 5,
	},
	["BumperRearPS"] = {
		partId = "DAMNBumperRear",
		itemToModel = {
			["Base.69miniBumperRear0"] = "BumperRear0",
			["Base.69miniBumperRear1"] = "BumperRear1",
		},
	},
	["Roofrack"] = {
		partId = "MINI69Roofrack",
		itemToModel = {
			["Base.69miniRoofrack1"] = "Roofrack0",
		},
		default = "trve_random",
		noPartChance = 75,
	},
	["DoorFrontLeftArmor"] = {
		partId = "DAMNFrontLeftArmor",
		itemToModel = {
			["Base.69miniFrontDoorArmor"] = "MINI69leftdoora",
		},
	},
	["DoorFrontRightArmor"] = {
		partId = "DAMNFrontRightArmor",
		itemToModel = {
			["Base.69miniFrontDoorArmor"] = "MINI69rightdoora",
		},
	},
	["WindowRearLeftArmor"] = {
		partId = "DAMNRearLeftArmor",
		itemToModel = {
			["Base.69miniRearWindowArmor"] = "MINI69leftwinra",
		},
	},
	["WindowRearRightArmor"] = {
		partId = "DAMNRearRightArmor",
		itemToModel = {
			["Base.69miniRearWindowArmor"] = "MINI69rightwinra",
		},
	},
	["WindshieldArmor"] = {
		partId = "DAMNWindshieldArmor",
		itemToModel = {
			["Base.69miniWindshieldArmor"] = "MINI69winda",
		},
	},
	["WindshieldRearArmor"] = {
		partId = "DAMNWindshieldRearArmor",
		itemToModel = {
			["Base.69miniWindshieldRearArmor"] = "MINI69windra",
		},
	},
	["SpareTire"] = {
		partId = "DAMNSpareTire",
		itemToModel = {
			["Base.69miniTire1"] = "SpareTire",
            ["damnCraft.SmallTire1"] = "SpareTireU",
		},
		default = "trve_random",
		noPartChance = 33,
	},
    ["TireFrontLeft"] = {
		partId = "TireFrontLeft",
		itemToModel = {
			["Base.69miniTire1"] = "miniTire",
            ["damnCraft.SmallTire1"] = "miniTireUNIL",
		},
	},
	["TireFrontRight"] = {
		partId = "TireFrontRight",
		itemToModel = {
			["Base.69miniTire1"] = "miniTire",
            ["damnCraft.SmallTire1"] = "miniTireUNIR",
		},
	},
	["TireRearLeft"] = {
		partId = "TireRearLeft",
		itemToModel = {
			["Base.69miniTire1"] = "miniTire",
            ["damnCraft.SmallTire1"] = "miniTireUNIL",
		},
	},
	["TireRearRight"] = {
		partId = "TireRearRight",
		itemToModel = {
			["Base.69miniTire1"] = "miniTire",
            ["damnCraft.SmallTire1"] = "miniTireUNIR",
		},
	},
});

MINI69.OnCreate = MINI69.OnCreate or {}

function MINI69.OnCreate.IllTakeThatBoxAlso(craftRecipeData, character)
    character:getInventory():AddItem("Base.69miniTeaCrate");
end