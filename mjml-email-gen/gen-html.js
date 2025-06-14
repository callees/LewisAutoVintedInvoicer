import mjml2html from 'mjml';

function generateItemMJML(itemName, itemPrice) {
  return `
<mj-section background-color="#ffffff" padding="5px 25px">
  <mj-column width="100%">
    <mj-text padding="0px" font-family="Helvetica, Arial, sans-serif" color="#000000">
      <table width="100%" style="border-collapse: collapse; font-size: 12px;">
        <tr>
          <td align="left" style="font-size: 12px;"><strong>${itemName}</strong></td>
          <td align="right" style="font-size: 12px;">${itemPrice}</td>
        </tr>
      </table>
    </mj-text>
  </mj-column>
</mj-section>
  `;
}

function generateItemsMJML(boughtItems) {
  return boughtItems.map(item =>
    generateItemMJML(item.ItemName, item.ItemPrice)
  ).join('');
}

const orderDetails = process.argv.slice(2);
const [orderNumber, orderDate, vintedUsername, rawItems, deliveryPrice, protectionProPrice, totalPrice] = orderDetails;
const boughtItems = JSON.parse(rawItems);
const itemMJML = generateItemsMJML(boughtItems);

const mjmlString = `
<mjml>
  <mj-body background-color="#b5cccb">
    <mj-section background-color="#ffffff" padding-bottom="20px" padding-top="20px">
    <mj-column width="100%">
      <mj-image src="https://i.ibb.co/mCtHch0w/dealorean-logo.png" alt="" align="center" border="none" width="150px" padding-left="0px" padding-right="0px" padding-bottom="0px" padding-top="0"></mj-image>
    </mj-column>
    </mj-section>
<mj-section background-color="#ffffff" padding="10px 25px">
  <mj-column width="100%">
    <mj-text font-size="16px" color="#000000" font-family="Helvetica, Arial, sans-serif" align="center" padding="10px 0">
      🚗💨 <strong>Welcome to the Future, ${vintedUsername}!</strong><br/>
    </mj-text>
  </mj-column>
</mj-section>
    
<mj-section padding="5px 0 5px 0">
  <mj-column width="100%">
    <mj-text font-size="15px" color="#3a3a3a" font-family="Helvetica, Arial, sans-serif" align="center" padding="10px 0">
      Great Scott! You’ve jumped through time to score deals so good, even Doc Brown would be impressed. Your future-self just made a smart move!
    </mj-text>
  </mj-column>
</mj-section>
    
  ${itemMJML}

    <mj-section background-color="#ffffff" padding="5px 25px">
      <mj-column width="100%">
        <mj-text padding="0px" font-family="Helvetica, Arial, sans-serif" color="#000000">
          <table width="100%" style="border-collapse: collapse; font-size: 11px;">
            <tr>
              <td align="left">Delivery</td>
              <td align="right">${deliveryPrice}</td>
            </tr>
          </table>
        </mj-text>
      </mj-column>
    </mj-section>

    <mj-section background-color="#ffffff" padding="5px 25px">
      <mj-column width="100%">
        <mj-text padding="0px" font-family="Helvetica, Arial, sans-serif" color="#000000">
          <table width="100%" style="border-collapse: collapse; font-size: 12px; font-weight: bold;">
            <tr>
              <td align="left">Total</td>
              <td align="right">£${totalPrice}</td>
            </tr>
          </table>
        </mj-text>
      </mj-column>
    </mj-section>

 <mj-section background-color="#ffffff" padding="5px 25px" text-align="center">
</mj-section>
    <mj-section background-color="#0878ad" padding-left="25px" padding-bottom="20px" padding-top="20px">
        <mj-column width="50%">
    <mj-text font-size="12px" color="#ffffff" font-family="Ubuntu, Helvetica, Arial, sans-serif" padding="0">
      <strong>Order Number:</strong> ${orderNumber}
    </mj-text>
  </mj-column>
  <mj-column width="50%">
    <mj-text font-size="12px" color="#ffff" font-family="Ubuntu, Helvetica, Arial, sans-serif" padding="0">
      <strong>Order Date:</strong> ${orderDate}
    </mj-text>
  </mj-column>
    </mj-section>
    <mj-section background-color="#0878ad" padding-bottom="5px" padding-top="0">
      
      <mj-column width="100%">
        <mj-divider border-color="#ffffff" border-width="2px" border-style="solid" padding-left="20px" padding-right="20px" padding-bottom="0px" padding-top="0"></mj-divider>
      </mj-column>
    </mj-section>
  </mj-body>
</mjml>
`;

const htmlOutput = mjml2html(mjmlString);

// Remove all unnecessary \n and whitespace between HTML tags
const cleanedHTML = htmlOutput.html.replace(/\n/g, '').replace(/\s{2,}/g, ' ');

process.stdout.write(cleanedHTML);
