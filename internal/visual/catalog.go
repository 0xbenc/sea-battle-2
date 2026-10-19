package visual

type Scene struct {
	ID, Title, Category, Camera string
	Kind, Ship, Team, Weather   int
	Duration                    float64
	Line                        string
}

const (
	Arrival = iota
	Tactical
	Portrait
	Launch
	Miss
	Hit
	Sink
	Result
	Broadcast
	Depth
	Fleet
	Duel
	AirOps
	Scope
	Spectacle
)

var Scenes = []Scene{
	{"arrival", "THE WATER BELONGS TO NOBODY", "OPENING", "SIDE", Arrival, 4, 0, 2, 12, "That has not stopped either party from claiming it."},
	{"human-station", "GENERAL QUARTERS", "COMMAND", "TOP", Tactical, 3, 0, 0, 12, "All stations report unnecessarily enthusiastic."},
	{"alien-station", "THE HUNGER BELOW", "COMMAND", "TOP", Tactical, 8, 1, 3, 12, "The ocean has been classified as soup."},
	{"deployment", "FLEET DISPOSITION", "PLACEMENT", "TOP", Tactical, 4, 0, 0, 10, "Five vessels. Seventeen squares. No refunds."},
	{"cyclone", "SMALL BOAT. LARGE OPINIONS.", "FLEET / A", "SIDE", Portrait, 0, 0, 0, 10, "Cyclone Class Patrol Boat / 2 spaces"},
	{"virginia", "SILENCE IS A WEAPONS SYSTEM", "FLEET / A", "SIDE", Portrait, 1, 0, 1, 10, "Virginia Class Attack Submarine / 3 spaces"},
	{"freedom", "SHALLOW WATER. DEEP CONCERNS.", "FLEET / A", "SIDE", Portrait, 2, 0, 0, 10, "Freedom Class Littoral Combat Ship / 3 spaces"},
	{"zumwalt", "LOW OBSERVABILITY. HIGH ATTITUDE.", "FLEET / A", "SIDE", Portrait, 3, 0, 2, 10, "Zumwalt Class Destroyer / 4 spaces"},
	{"ford", "FIVE SQUARES OF FOREIGN POLICY", "FLEET / A", "SIDE", Portrait, 4, 0, 0, 10, "Gerald R. Ford Class Aircraft Carrier / 5 spaces"},
	{"skiff", "THE SMALL ONE ALSO BITES", "FLEET / B", "SIDE", Portrait, 5, 1, 3, 10, "Drone Skiff / 2 spaces"},
	{"lurker", "DO NOT LOOK UNDER THE WATER", "FLEET / B", "SIDE", Portrait, 6, 1, 1, 10, "Deep Lurker / 3 spaces"},
	{"ravager", "SHORE LEAVE HAS BEEN CANCELLED", "FLEET / B", "SIDE", Portrait, 7, 1, 3, 10, "Littoral Ravager / 3 spaces"},
	{"carapace", "HEAVILY ARMORED. POORLY SOCIALIZED.", "FLEET / B", "SIDE", Portrait, 8, 1, 2, 10, "Siege Carapace / 4 spaces"},
	{"hive", "MOTHER WOULD LIKE A WORD", "FLEET / B", "SIDE", Portrait, 9, 1, 3, 10, "Hive Carrier / 5 spaces"},
	{"human-bow", "HEAD ON / STEEL", "CAMERA STUDY", "FRONT", Portrait, 3, 0, 1, 10, "A narrow silhouette is still a large problem."},
	{"alien-bow", "HEAD ON / TEETH", "CAMERA STUDY", "FRONT", Portrait, 9, 1, 3, 10, "Please keep all limbs inside the hemisphere."},
	{"human-plan", "FLIGHT DECK / PLAN VIEW", "CAMERA STUDY", "TOP", Portrait, 4, 0, 0, 10, "Top-down recognition study / wings folded"},
	{"alien-plan", "BROOD DECK / PLAN VIEW", "CAMERA STUDY", "TOP", Portrait, 9, 1, 3, 10, "Top-down recognition study / eggs armed"},
	{"launch", "SEND OUR REGARDS", "FIRE CONTROL", "SIDE", Launch, 3, 0, 2, 9, "Diplomatic channel: 155 millimeters."},
	{"spit", "BIOLOGICAL COUNTERARGUMENT", "FIRE CONTROL", "SIDE", Launch, 7, 1, 3, 9, "The ammunition is experiencing separation anxiety."},
	{"miss", "AGGRESSIVE OCEANOGRAPHY", "OUTCOME / MISS", "SIDE", Miss, 8, 0, 0, 8, "No contact. The water has learned its lesson."},
	{"hit", "CONTACT. UNFORTUNATELY.", "OUTCOME / HIT", "SIDE", Hit, 8, 0, 2, 9, "Confirmed hit / enemy vessel remains afloat"},
	{"retaliation", "HULL WARRANTY VOID", "OUTCOME / HIT", "SIDE", Hit, 2, 1, 1, 9, "Acid damage / crew requests less contact"},
	{"sinking", "ONE LESS MOUTH TO FEED", "OUTCOME / SUNK", "SIDE", Sink, 9, 0, 3, 13, "Hive Carrier destroyed / 5 spaces resolved"},
	{"loss", "STEEL RETURNS TO THE SEA", "OUTCOME / SUNK", "SIDE", Sink, 4, 1, 2, 13, "Gerald R. Ford lost / all hands, abandon ship"},
	{"depth", "SOMETHING MOVED", "UNDERWATER", "SIDE", Depth, 6, 1, 1, 12, "Passive sonar / contact is chewing"},
	{"storm", "BAD WEATHER. WORSE NEIGHBORS.", "ENVIRONMENT", "FRONT", Portrait, 4, 0, 2, 12, "Storm sea / bow-on silhouette and rain study"},
	{"victory", "OCEAN LIBERATED", "AFTERMATH", "SIDE", Result, 4, 0, 0, 12, "The ocean has declined to comment."},
	{"defeat", "NUTRITIONAL VICTORY", "AFTERMATH", "SIDE", Result, 9, 1, 3, 12, "Human fleet: locally sourced. Sustainably terrified."},
	{"bulletin", "FEDERAL MARITIME BULLETIN", "TRANSMISSION", "FRONT", Broadcast, 0, 0, 0, 10, "REMEMBER: BUGS CANNOT FILE A MARITIME CLAIM."},
	{"hivecast", "A MESSAGE FROM THE BROOD", "TRANSMISSION", "FRONT", Broadcast, 9, 1, 3, 10, "YOUR VESSEL HAS BEEN ADDED TO THE MENU."},
	{"human-fleet", "THE SEVENTEEN-SQUARE NAVY", "FLEET FORMATION", "TOP", Fleet, 4, 0, 0, 12, "Five distinct silhouettes / United States Navy fleet"},
	{"alien-fleet", "THE FOOD CHAIN HAS ARRIVED", "FLEET FORMATION", "TOP", Fleet, 9, 1, 3, 12, "Five distinct organisms / alien fleet"},
	{"exchange", "A BRIEF EXCHANGE OF VIEWS", "CINEMATIC SEQUENCE", "SIDE", Duel, 3, 0, 2, 17, "Gun director > launch > pursuit > impact > confirmed result"},
	{"counterexchange", "THE BROOD WOULD LIKE TO REPLY", "CINEMATIC SEQUENCE", "SIDE", Duel, 7, 1, 3, 17, "Synaptic lock > ejection > pursuit > impact > confirmed result"},
	{"air-wing", "EXPEDITED CUSTOMER SERVICE", "FLIGHT OPERATIONS", "TOP", AirOps, 4, 0, 0, 12, "Air wing deployed / special delivery, same square"},
	{"brood-wing", "THE CHILDREN ARE HUNGRY", "BROOD OPERATIONS", "TOP", AirOps, 9, 1, 3, 12, "Brood released / express digestion available"},
	{"periscope", "A VERY SMALL WINDOW OF OPPORTUNITY", "OPTICAL CONTACT", "SIDE", Scope, 9, 0, 1, 12, "Virginia / periscope depth / enemy silhouette study"},
	{"overkill", "A PERFECTLY PROPORTIONATE RESPONSE", "CINEMATIC EXCESS", "SIDE", Spectacle, 5, 0, 2, 14, "One grid square. An entire procurement budget."},
	{"digestive-event", "THE SEA IS EXPERIENCING INDIGESTION", "CINEMATIC EXCESS", "SIDE", Spectacle, 0, 1, 3, 14, "The brood insists this is a normal bodily function."},
}

func Find(id string) int {
	for i, s := range Scenes {
		if s.ID == id {
			return i
		}
	}
	return -1
}

type State struct {
	ShotMiss, Slow             bool
	Confirmed                  bool
	Scene                      int
	T                          float64
	SD                         bool
	Weather                    int
	Camera                     string
	Paused, Tour, Help, Picker bool
	Selection                  int
	TargetX, TargetY           int
	Alt                        bool
}

func DefaultState() State { return State{Weather: -1, Tour: true, TargetX: 6, TargetY: 4} }
func (s State) Size() string {
	if s.SD {
		return "SD"
	}
	return "HD"
}
