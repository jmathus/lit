package main

import ("strings"
  "fmt"
  "unicode"
  "net/http"
  "io/ioutil"
  "strconv"
  "log")

func getCryptocurrencyPrice(desiredCurrency string) string {
  // Connects to online website with cryptocurrency prices.
  resp, err := http.Get("https://coinranking.com/")
  if err != nil {
    log.Fatal(err)
  }

  // Closes the connection to the website after all of our code runs.
  defer resp.Body.Close()

  body, err := ioutil.ReadAll(resp.Body)
  if err != nil{
    log.Fatal(err)
  }

  // These are the two html tags that we look for to find the names of the cryptocurrencies
  // and their respective real time prices.
  /*<div class="coin-list__body__row__cryptocurrency__name"><span class="coin-name">Bitcoin</span></div>
  <span class="coin-list__body__row__price__value">14,568.24</span>*/

  // Converts the body of the website we accessed from bytes into a string we can work with.
  htmlBodyString := string(body)

  // Split the body string based on the cryptocurrency name tags and extracts the names alone.
  currencyNamesListPlus := strings.Split(htmlBodyString, "<div class=\"coin-list__body__row__cryptocurrency__name\"><span class=\"coin-name\">")
  var currencyNamesList []string
  for _, element := range currencyNamesListPlus {
    currency := strings.Split(element, "<")[0]
    currencyNamesList = append(currencyNamesList, currency)
  }

  // Split the body string based on the cryptocurrency price tags and extracts the prices alone.
  currencyValuesListPlus := strings.Split(htmlBodyString, "<span class=\"coin-list__body__row__price__value\">")
  var currencyValuesList []string
  for _, element := range currencyValuesListPlus {
    value := strings.Split(element, "<")[0]
    currencyValuesList = append(currencyValuesList, value)
  }

  // Makes sure that the desiredCurrency is in the correct format (First letter capitalized, the rest not)
  runeArr := []rune(desiredCurrency)
  formattedDesiredCurrency := ""

  for i := range runeArr {
    if i == 0 {
      formattedDesiredCurrency += string(unicode.ToUpper(runeArr[i]))
    } else{
      formattedDesiredCurrency += string(unicode.ToLower(runeArr[i]))
    }
  }

  // Prints the list of cryptocurrencies and their corresponding prices.
  for index, _ := range currencyNamesList{
    if currencyNamesList[index] == formattedDesiredCurrency {
      return currencyValuesList[index]
    }
  }

  return "None Available"
}

func getCryptocurrencyExchangeRate(desiredCurrency1 string, desiredCurrency2 string) string {
  if (desiredCurrency1 == "None") || (desiredCurrency2 == "None") {
    return "None"
  }
  value1String := strings.Join(strings.Split(getCryptocurrencyPrice(desiredCurrency1), ","), "")
  value2String := strings.Join(strings.Split(getCryptocurrencyPrice(desiredCurrency2), ","), "")

  value1, err1 := strconv.ParseFloat(value1String, 64)
  value2, err2 := strconv.ParseFloat(value2String, 64)
  if err1 != nil {
    fmt.Println(err1)
    return "Error"
  }
  if err2 != nil {
    fmt.Println(err2)
    return "Error"
  }

  conversion := value1/value2
  conversionString := strconv.FormatFloat(conversion, 'f', 2, 32)
  stringValue := "Exchange rate is about 1 " + desiredCurrency1 + " for " + conversionString + " " + desiredCurrency2

  return stringValue
}
