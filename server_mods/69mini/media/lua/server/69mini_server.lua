require "DAMN_Parts";
require "DAMN_Spawns";

--***********************************************************
--**                   KI5 / bikinihorst                   **
--***********************************************************

DAMN.Parts:processConfigV2("MINI69", {
	["BumperFront"] = {
		partId = "MINI69BumperFront",
		itemToModel = {
			["Base.69miniBumperFront0"] = "FrontBumper0",
			["Base.69miniBumperFront1"] = "FrontBumper1",
			["Base.69miniBumperFront2"] = "FrontBumper2",
		},
		default = "trve_random",
		noPartChance = 5,
	},
	["BumperFrontIJ"] = {
		partId = "MINI69BumperFront",
		itemToModel = {
			["Base.69miniBumperFront2"] = "FrontBumper2",
			["Base.69miniBumperFront0"] = "FrontBumper0",
			["Base.69miniBumperFront1"] = "FrontBumper1",
		},
		default = "first",
	},
	["BumperFrontPS"] = {
		partId = "MINI69BumperFront",
		itemToModel = {
			["Base.69miniBumperFront0"] = "FrontBumper0",
			["Base.69miniBumperFront1"] = "FrontBumper1",
			["Base.69miniBumperFront2"] = "FrontBumper2",
		},
	},
	["BumperRear"] = {
		partId = "MINI69BumperRear",
		itemToModel = {
			["Base.69miniBumperRear0"] = "BumperRear0",
			["Base.69miniBumperRear1"] = "BumperRear1",
		},
		default = "trve_random",
		noPartChance = 5,
	},
	["BumperRearPS"] = {
		partId = "MINI69BumperRear",
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
		partId = "MINI69DoorFrontLeftArmor",
		itemToModel = {
			["Base.69miniFrontDoorArmor"] = "MINI69leftdoora",
		},
	},
	["DoorFrontRightArmor"] = {
		partId = "MINI69DoorFrontRightArmor",
		itemToModel = {
			["Base.69miniFrontDoorArmor"] = "MINI69rightdoora",
		},
	},
	["WindowRearLeftArmor"] = {
		partId = "MINI69WindowRearLeftArmor",
		itemToModel = {
			["Base.69miniRearWindowArmor"] = "MINI69leftwinra",
		},
	},
	["WindowRearRightArmor"] = {
		partId = "MINI69WindowRearRightArmor",
		itemToModel = {
			["Base.69miniRearWindowArmor"] = "MINI69rightwinra",
		},
	},
	["WindshieldArmor"] = {
		partId = "MINI69WindshieldArmor",
		itemToModel = {
			["Base.69miniWindshieldArmor"] = "MINI69winda",
		},
	},
	["WindshieldRearArmor"] = {
		partId = "MINI69WindshieldRearArmor",
		itemToModel = {
			["Base.69miniWindshieldRearArmor"] = "MINI69windra",
		},
	},
	["SpareTire"] = {
		partId = "MINI69SpareTire",
		itemToModel = {
			["Base.69miniTire1"] = "SpareTire",
		},
		default = "trve_random",
		noPartChance = 33,
	},
});


function MINI69.ContainerAccess.TruckBed(vehicle, part, chr)
	if chr:getVehicle() == vehicle then
		local seat = vehicle:getSeat(chr)
		return seat == 3 or seat == 2 or seat == 1 or seat == 0;
	elseif chr:getVehicle() then
		return false
	else
		if not vehicle:isInArea(part:getArea(), chr) then return false end
		local doorPart = vehicle:getPartById("TrunkDoor")
		if doorPart and doorPart:getDoor() and not doorPart:getDoor():isOpen() then
			return false
		end
		return true
	end
end

function MINI69.ContainerAccess.Roofrack(vehicle, part, chr)
	if chr:getVehicle() then return false end
	if not vehicle:isInArea(part:getArea(), chr) then return false end
	return true
end

function Recipe.OnCreate.IllTakeThatBoxAlso(items, result, player)
    player:getInventory():AddItem("Base.69miniTeaCrate");
end