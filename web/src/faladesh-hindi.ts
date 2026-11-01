import copy from "./locales/faladesh-simple-hi.json" with {type:"json"};

const replacements=Object.entries(copy).sort((a,b)=>b[0].length-a[0].length);
const subjects:Record<string,string>={"स्वभाव":"आपके स्वभाव में","पढ़ाई":"पढ़ाई में","करियर":"काम में","धन और बचत":"पैसों के मामले में","संबंध":"रिश्ते में","घर-परिवार":"घर-परिवार में","दिनचर्या और विश्राम":"रोज की जिंदगी में"};

/** simpleFaladeshHindi rewrites retained editorial wording for display without changing findings, certainty, dates or their evidence. */
export function simpleFaladeshHindi(text:string){
  let result=text;
  for(const [original,simple] of replacements)result=result.split(original).join(simple);
  return result.replace(/^(स्वभाव|पढ़ाई|करियर|धन और बचत|संबंध|घर-परिवार|दिनचर्या और विश्राम) में (.*?) जैसे पहलू महत्वपूर्ण हो सकते हैं।/u,(_match,subject:string,themes:string)=>`${subjects[subject]} ये बातें जरूरी हो सकती हैं: ${themes}।`);
}
