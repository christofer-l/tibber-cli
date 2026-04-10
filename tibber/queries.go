package tibber

const HomesQuery = `{
  viewer {
    homes {
      id
      appNickname
      address {
        address1
        postalCode
        city
        country
      }
    }
  }
}`

const ConsumptionQuery = `query ($last: Int!) {
  viewer {
    homes {
      consumption(resolution: HOURLY, last: $last) {
        nodes {
          from
          to
          cost
          unitPrice
          unitPriceVAT
          consumption
          consumptionUnit
        }
      }
    }
  }
}`

const PricesQuery = `{
  viewer {
    homes {
      currentSubscription {
        priceInfo {
          current {
            total
            energy
            tax
            startsAt
            currency
            level
          }
          today {
            total
            energy
            tax
            startsAt
            currency
            level
          }
          tomorrow {
            total
            energy
            tax
            startsAt
            currency
            level
          }
        }
      }
    }
  }
}`

const DailyConsumptionQuery = `query ($last: Int!) {
  viewer {
    homes {
      consumption(resolution: DAILY, last: $last) {
        nodes {
          from
          to
          cost
          unitPrice
          unitPriceVAT
          consumption
          consumptionUnit
        }
      }
    }
  }
}`
