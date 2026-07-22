require "DAMN_Armor_Shared";

--***********************************************************
--**                   KI5 / bikinihorst                   **
--***********************************************************
--v2.0.0

MINI69 = MINI69 or {};

function MINI69.activeArmor(player, vehicle)
   
		--

			local protection = vehicle:getPartById("MINI69BumperFront")
			local inventoryItem = protection:getInventoryItem();
			local part = vehicle:getPartById("EngineDoor")
				if part and protection and part:getInventoryItem() and inventoryItem and part:getModData()
				then 
					if inventoryItem:getFullType() ~= "Base.69miniBumperFront2" then
						local partCond = tonumber(part:getModData().saveCond)
						if protection:getCondition() > 0 and partCond
						then
							if part:getCondition() < partCond
							then
								DAMN.Armor:setPartCondition(part, partCond);
								local cond = protection:getCondition() - ZombRandBetween(0,15);
								DAMN.Armor:setPartCondition(protection, cond);
							end
						end
					else
						local partCond = tonumber(part:getModData().saveCond)
						if protection:getCondition() > 0 and partCond
						then
							if part:getCondition() < partCond
							then
								DAMN.Armor:setPartCondition(part, partCond);
								local cond = protection:getCondition() - (ZombRandBetween(0,100) <= 55 and ZombRandBetween(0,3) or 0);
								DAMN.Armor:setPartCondition(protection, cond);
							end
						end
				end
			end

		--

			local protection = vehicle:getPartById("MINI69BumperRear")
			local inventoryItem = protection:getInventoryItem();
			local part = vehicle:getPartById("TrunkDoor")
				if part and protection and inventoryItem and part:getModData()
				then 
					local partCond = tonumber(part:getModData().saveCond)
					if protection:getCondition() > 0 and partCond
					then
						if part:getCondition() < partCond
						then
							DAMN.Armor:setPartCondition(part, partCond);
							local cond = protection:getCondition() - ZombRandBetween(1,10);
							DAMN.Armor:setPartCondition(protection, cond);
						end
					end
				end	

		--

			for partId, armorPartId in pairs({
				["WindowFrontLeft"] = "MINI69DoorFrontLeftArmor",
				["WindowFrontRight"] = "MINI69DoorFrontRightArmor",
                ["WindowRearLeft"] = "MINI69WindowRearLeftArmor",
				["WindowRearRight"] = "MINI69WindowRearRightArmor",
			}) do
				local part = vehicle:getPartById(partId);
				local protection = vehicle:getPartById(armorPartId);
				if protection and protection:getInventoryItem() and part and part:getModData()
				then
					local partCond = tonumber(part:getModData().saveCond);
					if protection:getCondition() > 0 and partCond and part:getCondition() < partCond
					then
						DAMN.Armor:setPartCondition(part, partCond);
                        local cond = protection:getCondition() - ZombRandBetween(0,5)
						DAMN.Armor:setPartCondition(protection, cond);
					end
				end
			end

		--

			for partId, armorPartId in pairs({
				["HeadlightLeft"] = "MINI69BumperFront",
				["HeadlightRight"] = "MINI69BumperFront",
				["HeadlightRearLeft"] = "MINI69BumperRear",
				["HeadlightRearRight"] = "MINI69BumperRear",
			}) do
				local part = vehicle:getPartById(partId);
				local protection = vehicle:getPartById(armorPartId);
				if protection and protection:getInventoryItem() and part and part:getModData()
				then
					local partCond = tonumber(part:getModData().saveCond);
					if protection:getCondition() > 0 and partCond and part:getCondition() < partCond
					then
						DAMN.Armor:setPartCondition(part, partCond);
					end
				end
			end

		--

			local protection = vehicle:getPartById("MINI69WindshieldArmor")
			local part = vehicle:getPartById("Windshield")
			if protection and protection:getInventoryItem() and part and part:getModData()
			then
				local partCond = tonumber(part:getModData().saveCond)
				if protection:getCondition() > 0 and partCond
				then
					if part:getCondition() < partCond
					then
						DAMN.Armor:setPartCondition(part, partCond);
						local cond = protection:getCondition() - (ZombRandBetween(0,100) <= 65 and ZombRandBetween(0,3) or 0)
						DAMN.Armor:setPartCondition(protection, cond);
					end
				end
			end

		--

		for i, freezeState in ipairs ({"MINI69SpareTire", "MINI69Roofrack"})
				do
					if vehicle:getPartById(freezeState) then
						local part = vehicle:getPartById(freezeState)
						local freezeCond = tonumber(part:getModData().saveCond)
					    	if freezeCond and part:getCondition() < freezeCond then
					    		DAMN.Armor:setPartCondition(part, freezeCond);
							end
					end
			end

		--

			local protection = vehicle:getPartById("MINI69WindshieldRearArmor")
			local part = vehicle:getPartById("WindshieldRear")
			if protection and protection:getInventoryItem() and part and part:getModData()
			then
				local partCond = tonumber(part:getModData().saveCond)
				if protection:getCondition() > 0 and partCond
				then
					if part:getCondition() < partCond
					then
						DAMN.Armor:setPartCondition(part, partCond);
						local cond = protection:getCondition() - (ZombRandBetween(0,100) <= 65 and ZombRandBetween(0,3) or 0)
						DAMN.Armor:setPartCondition(protection, cond);
					end
				end
			end

end

DAMN.Armor:add("Base.69mini", MINI69.activeArmor);
DAMN.Armor:add("Base.69miniIJ", MINI69.activeArmor);
DAMN.Armor:add("Base.69miniMrB", MINI69.activeArmor);
DAMN.Armor:add("Base.69miniPS", MINI69.activeArmor);
DAMN.Armor:add("Base.69miniUnionJack", MINI69.activeArmor);