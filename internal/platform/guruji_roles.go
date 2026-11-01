package platform

// gurujiSignLabel exposes the primary status while retaining overlapping own-sign status in readable evidence.
func gurujiSignLabel(status GurujiSignStatus) GurujiWording {
	label := map[string]GurujiWording{"exalted": {"उच्च", "Exalted"}, "debilitated": {"नीच", "Debilitated"}, "own": {"स्वगृही", "Own sign"}, "other": {"अन्य", "Other"}}[status.Primary]
	if status.Exalted && status.OwnSign {
		label.Hindi += " (स्वगृही भी)"
		label.English += " (also own sign)"
	}
	return label
}

// gurujiSignStatus evaluates the board's sign numbers independently of house and preserves simultaneous exaltation and ownership.
func gurujiSignStatus(planet string, sign int) GurujiSignStatus {
	out := GurujiSignStatus{Primary: "other"}
	if sign < 1 || sign > 12 {
		return out
	}
	dignity := gurujiDignity(planet, sign)
	out.Exalted, out.Debilitated = dignity == "exalted", dignity == "debilitated"
	owners := []string{"Mars", "Venus", "Mercury", "Moon", "Sun", "Mercury", "Venus", "Mars", "Jupiter", "Saturn", "Saturn", "Jupiter"}
	out.OwnSign = owners[sign-1] == planet
	switch {
	case out.Exalted:
		out.Primary = "exalted"
	case out.Debilitated:
		out.Primary = "debilitated"
	case out.OwnSign:
		out.Primary = "own"
	}
	return out
}

// gurujiRoleEffect combines the immutable board role with sign capacity only; it makes no conjunction, aspect, cancellation, timing or final-outcome claim.
func gurujiRoleEffect(role string, status GurujiSignStatus) GurujiRoleEffect {
	if role == "karak" {
		switch {
		case status.Exalted:
			return GurujiRoleEffect{"support", GurujiWording{"यह ग्रह लग्न के लिए कारक है और उच्च राशि में है। इससे शुभ फल देने की क्षमता को मजबूती मिलती है; अंतिम फल केवल इससे तय नहीं होता।", "This planet is Karak for the ascendant and exalted by sign. Its capacity to give favourable results is strengthened; this alone does not determine the final outcome."}}
		case status.Debilitated:
			return GurujiRoleEffect{"attention", GurujiWording{"यह ग्रह लग्न के लिए कारक है, लेकिन नीच राशि में होने से शुभ फल देने की क्षमता कमजोर या बाधित हो सकती है। इसकी भूमिका कारक ही रहती है।", "This planet remains Karak for the ascendant. Debilitation may weaken or obstruct its capacity to give favourable results without changing its role to Marak."}}
		case status.OwnSign:
			return GurujiRoleEffect{"support", GurujiWording{"यह ग्रह लग्न के लिए कारक है और अपनी राशि में है। इससे अपनी भूमिका निभाने को सहारा मिलता है; अंतिम फल केवल इससे तय नहीं होता।", "This planet is Karak for the ascendant and in its own sign. This supports its capacity to perform its role without settling the final outcome."}}
		default:
			return GurujiRoleEffect{"context", GurujiWording{"बोर्ड के अनुसार यह ग्रह लग्न के लिए कारक है। इस राशि में उच्च, नीच या स्वगृही होने का विशेष संकेत नहीं है।", "The board classifies this planet as Karak for the ascendant. This sign is neither its exaltation, debilitation nor own sign."}}
		}
	}
	if role == "marak" {
		switch {
		case status.Exalted:
			return GurujiRoleEffect{"context", GurujiWording{"बोर्ड के अनुसार यह ग्रह लग्न के लिए मारक है और राशि से उच्च है। उच्च होने से इसकी भूमिका कारक नहीं बनती; इसे केवल उच्च होने के कारण शुभ फल का आधार नहीं माना गया है।", "The board classifies this planet as Marak; it is exalted by sign. Exaltation does not make it Karak and is not counted as automatic support for a favourable outcome."}}
		case status.Debilitated:
			return GurujiRoleEffect{"context", GurujiWording{"बोर्ड के अनुसार यह ग्रह मारक है और राशि से नीच है। इन दोनों बातों से अकेले अच्छा या बुरा अंतिम फल तय नहीं किया गया है।", "The board classifies this planet as Marak and its sign is debilitated. These facts alone are not used to declare a favourable or adverse final outcome."}}
		default:
			return GurujiRoleEffect{"context", GurujiWording{"बोर्ड के अनुसार इस ग्रह की भूमिका मारक है। राशि-स्थिति अलग रखी गई है; स्वगृही होने से भी भूमिका कारक नहीं बनती।", "The board classifies this planet as Marak. Its sign condition is separate; even an own-sign placement does not change its role to Karak."}}
		}
	}
	if role == "neutral" {
		return GurujiRoleEffect{"context", GurujiWording{"बोर्ड के अनुसार यह ग्रह लग्न के लिए सम है। उच्च या नीच राशि होने पर भी भूमिका सम ही रहती है; केवल राशि-स्थिति से शुभ या अशुभ अंतिम फल नहीं चुना गया है।", "The board classifies this planet as neutral for the ascendant. Exaltation or debilitation leaves that role unchanged and does not alone select a favourable or adverse outcome."}}
	}
	return GurujiRoleEffect{"context", GurujiWording{"इस ग्रह की लग्न-भूमिका बोर्ड में नहीं दी गई है। इसे अनुमान से कारक, मारक या सम नहीं बनाया गया है।", "The board does not list an ascendant role for this planet. No Karak, Marak or neutral classification has been inferred."}}
}
