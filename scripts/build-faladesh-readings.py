"""Build concise, bilingual reading summaries from the supplied conditional passages."""

import json
import pathlib
import re

# English is an original concise paraphrase of the supplied material, not a claimed shloka translation.
# Every row retains both possibilities; no Karak/Marak label selects one as a final outcome.
ENGLISH = {
    "Sun": [
        "Identity and the wish to lead can be prominent in your approach.|Confidence and initiative can help you take responsibility.|Leave room for advice when pride or impatience takes over.",
        "Recognition may feel closely connected to family, speech and financial security.|Clear communication can help you take a responsible family role.|A forceful tone or spending to maintain appearances can create strain.",
        "Your own effort and the way you express ideas can become important sources of identity.|Initiative, communication and sustained practice can develop your skills.|Avoid turning differences with siblings or colleagues into contests of authority.",
        "Home and a sense of personal authority may be closely connected.|Taking responsibility can support the home and its practical needs.|The need to control domestic decisions can disturb peace.",
        "You may seek recognition through learning, ideas and creative work.|Independent thinking and leadership can support your studies and expression.|Pride in your own views can make learning from others difficult.",
        "Responsibility, competition and solving everyday problems can stand out.|Discipline and persistence can help you face demanding work.|Power struggles and taking on too much can make the routine harder.",
        "Respect, independence and the balance of authority can matter in partnerships.|Clear boundaries and mutual recognition can support collaboration.|Trying to dominate decisions can make close relationships tense.",
        "Change and questions about shared resources may draw your attention.|Research and the ability to face uncertainty can become useful strengths.|Secrecy or struggles over control can complicate shared decisions.",
        "Beliefs, mentors and higher learning may shape how you find direction.|Commitment to study and principles can support your growth.|Rigid beliefs can make differences with teachers or family harder to discuss.",
        "Public responsibility, recognition and professional identity can be important themes.|Leadership and organised effort can help you take visible responsibility.|Authority conflicts and placing status above cooperation can become obstacles.",
        "Recognition may develop through networks, group work and long-term aims.|Leadership and useful contacts can support your goals.|Expectations of recognition can put pressure on friendships and collaboration.",
        "Private work, distant connections and the use of resources may need attention.|Quiet preparation and work away from the spotlight can be constructive.|Unplanned expenses or difficulty being recognised can be frustrating.",
    ],
    "Moon": [
        "You may be sensitive to the emotional tone of the people and places around you.|Empathy and adaptability can help you connect with others.|Changing moods or relying heavily on others' opinions can unsettle decisions.",
        "Family, the way you speak and a sense of financial security can carry emotional weight.|A caring tone can make others feel understood.|Emotional spending or speaking while upset can complicate family matters.",
        "Your feelings may find expression through communication, learning skills and close connections.|Imagination and responsiveness can support writing and communication.|Changing plans with each change of mood can interrupt steady practice.",
        "Home, family memories and a feeling of safety may strongly influence your peace of mind.|Care and emotional awareness can help create a welcoming home.|Domestic uncertainty can affect concentration and make reassurance more important.",
        "Learning, imagination and creative expression can be closely tied to your feelings.|Sensitivity can support creativity and understanding others.|Emotional reactions can make study or relationship expectations less steady.",
        "Your emotional state may respond closely to your routine and working environment.|Care and attentiveness can be useful in service and everyday responsibilities.|Repeated worry or reacting to small workplace tensions can be tiring.",
        "Close relationships and the responses of other people may matter deeply to you.|Understanding another person's needs can support cooperation.|Depending too much on a partner's mood can weaken your own steadiness.",
        "You may think deeply about uncertainty, change and matters that are not openly discussed.|Sensitivity and curiosity can support reflective study.|Repeatedly dwelling on uncertainty can make it harder to feel settled.",
        "Family values, beliefs and learning may carry a strong emotional connection.|An open and caring approach can help you learn from different people.|Taking differences of belief personally can create unnecessary strain.",
        "Work involving people and their changing needs may feel significant.|Adaptability can support public contact and service-oriented work.|Public reactions or changing moods can pull professional decisions off course.",
        "Friends, groups and your changing hopes may influence your sense of fulfilment.|Responsiveness can help you build supportive social connections.|Comparing your progress with others can lead to restless goals.",
        "Quiet time and an imaginative inner life may be important to you.|Reflection and work behind the scenes can feel nourishing.|Withdrawing for too long or spending to change a mood can become unhelpful.",
    ],
    "Mars": [
        "You may approach life with energy, independence and a wish to act quickly.|Courage and initiative can help you face challenges.|Impatience and responding before listening can turn small differences into conflict.",
        "Energy and directness may enter family discussions and financial decisions.|Decisive effort can help you take practical responsibility.|A sharp tone or acting impulsively with money can create friction.",
        "Effort, practical skills and the courage to begin can be central themes.|Persistence and hands-on practice can develop your abilities.|Competition with siblings or unnecessary risk can waste energy.",
        "You may want to take practical action over home, property and domestic responsibilities.|Energy can support repairs, building and managing practical needs.|Impatience or insisting on your own way can disturb peace at home.",
        "Your learning and creativity may have a competitive, active quality.|Enthusiasm can support focused projects and practical study.|Rushing decisions or taking speculative risks can undermine good preparation.",
        "You may meet everyday challenges with a strong competitive response.|Persistence and practical action can help you handle demanding tasks.|Turning every disagreement into a battle can exhaust you and others.",
        "Strong opinions and independence may be noticeable in close partnerships.|Direct effort can support shared work when both people have a voice.|Quick anger or trying to win every exchange can create relationship friction.",
        "Complex problems and sudden changes may draw out an intense response.|Courage and technical curiosity can support investigation.|Impulsive reactions to uncertainty or shared-money questions can increase tension.",
        "You may want to act on your beliefs rather than leave them as ideas.|Determination can support higher study and purposeful effort.|Arguments over principles can strain relationships with teachers or family.",
        "Action, technical work and the drive to achieve may be prominent in your work style.|Initiative and practical leadership can support demanding projects.|Haste and workplace conflict can interfere with sustained progress.",
        "Your goals and social connections may carry a strong competitive energy.|Determined effort can help you work towards concrete aims.|Impatience for gains or rivalry within groups can complicate cooperation.",
        "Energy may be directed towards distant places, private work or expenses.|Disciplined effort behind the scenes can be productive.|Restless spending or unresolved irritation can drain attention.",
    ],
    "Mercury": [
        "Curiosity, language and analysis may shape how you approach the world.|Learning and explaining ideas can become useful strengths.|Overthinking or changing your position too quickly can make decisions harder.",
        "Speech, practical knowledge and the management of resources may be important themes.|Thoughtful communication can support learning and financial organisation.|Clever words without consistency can weaken trust in family discussions.",
        "Communication and learning through repeated practice may come naturally into focus.|Writing, analysis and practical skills can develop through your own effort.|Scattered interests can interrupt the steady work needed for depth.",
        "Learning and discussion may be important to your experience of home.|Curiosity can support education and thoughtful domestic decisions.|Overanalysing family matters can make it harder to feel settled.",
        "Ideas, study and creative problem-solving may hold your attention.|Analysis and expression can support learning and intellectual work.|Restless attention or excessive calculation can interrupt deeper understanding.",
        "Solving practical problems and organising daily work may stand out.|Analysis and clear communication can support service and technical tasks.|Persistent worry or disputes over small details can complicate the routine.",
        "Conversation, negotiation and a meeting of minds can matter in partnerships.|Clear agreements and listening can support cooperation.|Overanalysing the relationship or using words evasively can create distance.",
        "You may be drawn to research, hidden details and complex questions.|Careful analysis can support investigation and deeper study.|Speculation and repeatedly searching for hidden meanings can create confusion.",
        "Learning, language and the exchange of ideas may help you find direction.|Curiosity can support higher study and constructive discussion.|Argument for its own sake can make guidance harder to receive.",
        "Communication and the practical use of knowledge can be prominent in your work.|Writing, analysis, commerce and organisation can suit this placement's themes.|Scattered professional plans or unclear communication can limit progress.",
        "Ideas and useful contacts may connect with your goals and income.|Communication and planning can help you participate in networks.|Too many shifting aims or overreliance on contacts can weaken follow-through.",
        "Private study, research and distant connections may interest you.|Quiet preparation and analytical work behind the scenes can be useful.|Mental restlessness or unclear spending plans can diffuse your attention.",
    ],
    "Jupiter": [
        "Knowledge, values and the wish to guide may shape your self-expression.|Generosity and a willingness to learn can support sound judgement.|Overconfidence or giving advice without listening can limit understanding.",
        "Family values, knowledge and the way resources are used may be significant.|Thoughtful speech and a learning-oriented family atmosphere can be helpful.|Excess or misplaced confidence can complicate financial expectations.",
        "Learning and guidance may find expression through communication and personal effort.|Thoughtful writing and sharing knowledge can develop through practice.|Advice needs to be supported by consistent action, not only good intentions.",
        "Education, family values and a sense of inner stability may matter to you.|Learning and considerate guidance can support the home.|Excessive expectations of domestic comfort can create dissatisfaction.",
        "Study, creativity and the wish to understand may be important themes.|Deep learning and careful guidance can be constructive strengths.|Certainty about your own ideas can make new perspectives harder to hear.",
        "Knowledge may be directed towards service and everyday problem-solving.|Teaching and thoughtful assistance can be useful in working life.|Taking on too many obligations or misjudging a dispute can create strain.",
        "Shared values, guidance and trust may be important in partnerships.|Respectful conversation can help both people learn and cooperate.|Idealising a partner or expecting agreement on every value can disappoint.",
        "You may seek understanding through research, change and shared responsibilities.|Reflection and patient study can support deeper insight.|Assuming that good intentions settle complex shared matters can be unhelpful.",
        "Higher learning, mentors and a search for meaning may be significant.|Study and openness to guidance can help you develop your principles.|Dogmatism can narrow the very understanding you are seeking.",
        "Knowledge, advice and responsibility may be visible in your work.|Teaching, guidance and ethical judgement can support professional contributions.|Expecting respect without staying practical can create professional difficulties.",
        "Learning, helpful networks and long-term aims may connect with each other.|Mentors and constructive groups can support your goals.|Expectations of easy gains or overextending your commitments can disappoint.",
        "Reflection, spiritual interests and giving may shape your private life.|Quiet study and generosity within your means can be constructive.|Unplanned giving or idealising distant opportunities can strain resources.",
    ],
    "Venus": [
        "Harmony, aesthetics and the way you relate to others may be important to your identity.|Warmth and creativity can help you build connections.|Relying too much on admiration or appearances can weaken your own priorities.",
        "Pleasant speech, family comfort and enjoyment may carry weight in your decisions.|Tact and an appreciation of quality can enrich family life.|Spending for comfort or avoiding difficult conversations can create problems.",
        "Creativity may find expression through communication and practical skills.|Artistic practice and a friendly manner can support expression.|Seeking praise more than sustained practice can interrupt development.",
        "Beauty, harmony and comfort may matter in your home environment.|Care and an aesthetic sense can help create a welcoming home.|Expensive expectations or avoiding domestic disagreements can create tension.",
        "Affection, imagination and creative enjoyment may be prominent themes.|Art and expressive learning can become satisfying outlets.|Idealised expectations in affection or pleasure can distract from responsibilities.",
        "Cooperation and the quality of daily relationships may matter in your work.|Tact can help with clients, service and workplace harmony.|Avoiding disagreement or overindulgence can make routine responsibilities harder.",
        "Affection and mutual enjoyment may be important in your partnerships.|Consideration and time together can support close connections.|Idealising a partner or expecting constant harmony can conceal unresolved differences.",
        "Intimacy, trust and shared resources may be significant themes.|Sensitivity can support understanding in close relationships.|Secrecy, dependence or unclear shared-money arrangements can create strain.",
        "Beauty, values and learning through different cultures may interest you.|Artistic study and thoughtful guidance can enrich your outlook.|Comfort or idealisation can sometimes take the place of deeper examination.",
        "Creativity and an understanding of people's preferences may enter your work.|Design, communication and social skills can support professional expression.|Depending too much on popularity can make professional choices less steady.",
        "Friendship, creative interests and social networks may support your aims.|Cooperation and artistic connections can open useful possibilities.|Pleasure-driven spending or unrealistic expectations of friends can be unhelpful.",
        "Private enjoyment, imagination and distant connections may be important themes.|Creative reflection and quiet time can be satisfying.|Unplanned spending on comfort or retreating from responsibilities can become costly.",
    ],
    "Saturn": [
        "You may approach life with seriousness and a strong awareness of responsibility.|Patience and sustained work can develop reliable strengths.|Self-doubt or excessive seriousness can make it harder to act confidently.",
        "Responsibility may be closely connected to family, speech and financial security.|Careful planning and consistency can support stability.|A restrained tone or persistent fear of scarcity can strain family communication.",
        "Skills and confidence may develop through repeated, steady effort.|Patience can support practical mastery and dependable communication.|Hesitation or distance from siblings can make collaboration more difficult.",
        "Home and family duties may carry a strong sense of responsibility.|Consistent care can support long-term domestic stability.|Feeling burdened or emotionally distant can reduce the comfort of home.",
        "Learning and creative expression may take a patient, disciplined form.|Persistence can help you work deeply on a subject.|Fear of mistakes or rigid expectations can restrict enjoyment and expression.",
        "Daily work, service and the ability to endure demands may stand out.|Consistency can help with difficult tasks and practical responsibilities.|Overwork and prolonged conflicts can make the routine feel heavy.",
        "Commitment and responsibility may be central to your expectations of partnerships.|Patience and reliability can support lasting cooperation.|Emotional distance or treating the relationship only as a duty can create strain.",
        "Complex changes and shared responsibilities may call for patience.|Careful investigation and endurance can help you handle uncertainty.|Prolonged worry or resisting necessary change can make the situation feel heavier.",
        "Beliefs and higher learning may develop through discipline and experience.|Patient study can help you examine your principles carefully.|Rigid beliefs or discouragement over delayed progress can narrow your outlook.",
        "Professional identity may be built through sustained responsibility.|Organisation and persistent effort can support dependable work.|Heavy obligations or slow recognition can make balance important.",
        "Long-term aims and practical networks may develop gradually.|Consistency and reliable connections can support lasting goals.|Feeling that progress is too slow can create unnecessary discouragement.",
        "Private responsibilities, distant work and expenses may need careful attention.|Disciplined preparation behind the scenes can be useful.|Isolation or worrying excessively about resources can become burdensome.",
    ],
    "Rahu": [
        "A wish to stand out or build a distinctive identity may be strong.|Unconventional thinking can help you explore unfamiliar possibilities.|Comparison, image-consciousness and haste can make direction less steady.",
        "Ambition may focus on financial security, family identity and the impact of your words.|An unusual perspective can support new ways of communicating and earning.|Unclear promises or the pursuit of status can weaken trust.",
        "Communication, initiative and the wish to try new approaches may stand out.|Experimentation can support practical skills and independent effort.|Shortcuts and attention-seeking can interfere with lasting competence.",
        "You may seek something different in your home, location or sense of belonging.|Adaptability can help you explore unfamiliar living arrangements.|Restlessness can make it difficult to feel satisfied with the home you have.",
        "Learning and creative expression may be drawn towards unusual ideas.|Experimentation can support inventive projects.|Speculation, distraction or exaggerated expectations can undermine preparation.",
        "Complex practical challenges may draw a resourceful response.|Persistence and unconventional problem-solving can help in competitive settings.|Manipulation or prolonged conflict can create avoidable complications.",
        "Partnerships may involve strong expectations or unfamiliar people and experiences.|Openness can help you learn from different perspectives.|Idealisation, unclear motives and rushed commitments can create confusion.",
        "Hidden questions, uncertainty and deep investigation may attract your attention.|Curiosity can support research into complex subjects.|Suspicion and taking poorly understood risks can increase unease.",
        "You may question established beliefs and seek unfamiliar teachers or cultures.|Independent inquiry can broaden your learning.|Following a dramatic claim without checking it can lead you away from sound guidance.",
        "Ambition, visibility and an unconventional professional path may be important themes.|New methods and an ability to work with changing environments can be useful.|Image-driven choices or shortcuts can undermine credibility.",
        "Large aims, broad networks and new sources of opportunity may draw your attention.|Adaptability can help you connect with diverse groups.|Pursuing gains without checking people and promises can create difficulties.",
        "Distant connections, private activity and unfamiliar environments may be significant.|Adaptability can support work away from familiar settings.|Hidden expenses or unclear arrangements can complicate your plans.",
    ],
    "Ketu": [
        "You may look inward and question the meaning of your identity and direction.|Reflection and independent study can deepen understanding.|Withdrawing or finding it hard to express yourself can create distance.",
        "You may feel detached from conventional ideas about family, status or savings.|Simple priorities and thoughtful speech can help clarify what matters.|Withdrawal from family discussion or neglect of practical resources can create strain.",
        "Your skills and communication may develop through independent, focused effort.|Concentration can support research and specialised practice.|Detachment from siblings or difficulty sharing ideas can limit cooperation.",
        "Inner contentment may matter more than the appearance of domestic comfort.|Reflection and simplicity can help you examine what home means to you.|Feeling disconnected from family or place can make belonging harder.",
        "Learning and creativity may take an inward, unconventional direction.|Deep concentration can support research and reflective work.|Doubt or emotional detachment can interrupt expression and enjoyment.",
        "You may prefer to solve practical problems in your own focused way.|Concentration and independence can support demanding tasks.|Neglecting routine details or withdrawing from cooperation can create complications.",
        "Close relationships may raise questions about closeness and independence.|Honesty and reflection can help clarify what you need from a partnership.|Emotional distance or unclear communication can leave the other person uncertain.",
        "Research, change and questions beneath the surface may strongly interest you.|Concentration can support deep investigation and reflection.|Isolation or assuming hidden meanings can make uncertainty harder to handle.",
        "You may question inherited beliefs and look for a more personal understanding.|Study and inward reflection can deepen your outlook.|Dismissing guidance too quickly can cut you off from useful learning.",
        "Meaning and independence may matter more than professional display.|Focused expertise can support work requiring depth.|Detachment from recognition or unclear direction can interrupt professional continuity.",
        "You may be selective about friends, groups and the goals worth pursuing.|Clear priorities can help you focus on meaningful connections.|Withdrawing from networks or losing interest in goals can limit practical support.",
        "Solitude, reflection and the wish to let go may be prominent themes.|Meditation and quiet study can provide constructive outlets.|Isolation or overlooking ordinary responsibilities can become unhelpful.",
    ],
}


def sentence(text):
    """Retain the first complete Hindi source sentence, including any stated condition."""
    value = re.sub(r"\s+", " ", text).strip(" •:\n")
    return value.split("।", 1)[0].strip() + "।"


def source_parts(passage):
    """Extract a source lead and both conditional branches without changing which one applies."""
    text = passage["hindi_excerpt"]
    positive = re.search(r"(?<!अ)शुभ (?:फल|प्रभाव)\s*:?(?:\s|\n)*|सकारात्मक फल\s*", text)
    negative = re.search(r"अशुभ या चुनौतीपूर्ण फल\s*|चुनौतीपूर्ण फल\s*|संभावित चुनौतियाँ\s*:?(?:\s|\n)*", text)
    if passage["planet"] == "Moon":
        positive = re.search(r"(?:यदि चंद्रमा|मजबूत चंद्रमा|चंद्रमा मजबूत)[^।]*", text)
        negative = re.search(r"(?:लेकिन )?(?:यदि )?चंद्रमा (?:पीड़ित|कमजोर)[^।]*", text)
    if passage["planet"] == "Mars" and passage["house"] in [5,7]:
        negative = re.search(r"(?:लेकिन )?मंगल पीड़ित होने पर",text)
    if not positive or not negative:
        raise ValueError(f"Missing polarity source: {passage['planet']} {passage['house']}")
    support_text = text[positive.end():] if passage["planet"] != "Moon" else text[positive.start():]
    caution_text = text[negative.start():] if passage["planet"] == "Moon" or (passage["planet"] == "Mars" and passage["house"] in [5,7]) else text[negative.end():]
    intro = text[:positive.start()].split("  ", 1)[-1]
    sentences = [re.sub(r"\s+", " ", item).strip() for item in intro.split("।") if item.strip()]
    # Prefer a statement linking the placement to its theme rather than the generic house definition or a later example.
    candidates = [item for item in sentences[:5] if re.search(r"इस भाव में|यहाँ|जब चंद्रमा|जब मंगल",item)]
    lead = (candidates[-1] if candidates else sentence(support_text)).removeprefix("दोस्तों, ")
    return {"meaning": lead.rstrip("।")+"।", "support": sentence(support_text), "caution": sentence(caution_text)}


def main():
    """Create the versioned reading copy; the importer attaches it only to new source snapshots."""
    root = pathlib.Path(__file__).resolve().parents[1]
    source = json.loads((root/"internal/platform/data/guruji-faladesh.json").read_text(encoding="utf-8"))
    hindi_leads = json.loads((root/"internal/platform/data/faladesh-hindi-leads.json").read_text(encoding="utf-8"))
    entries = []
    for passage in source["placements"]:
        planet, house = passage["planet"], passage["house"]
        hindi = source_parts(passage)
        hindi["meaning"] = hindi_leads[planet][house-1]
        english = ENGLISH[planet][house-1].split("|")
        item = {"planet":planet, "house":house}
        for key,en in zip(["meaning","support","caution"],english):
            item[key] = {"hi":hindi[key],"en":en}
        entries.append(item)
    result = {"version":"m4y-source-reading-v2", "entries":entries}
    (root/"internal/platform/data/faladesh-reading-copy.json").write_text(json.dumps(result,ensure_ascii=False,indent=2)+"\n",encoding="utf-8")
    print(f"Built {len(entries)} bilingual source-based reading summaries.")


if __name__ == "__main__":
    main()
