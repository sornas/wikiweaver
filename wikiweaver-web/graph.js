// The graph
var webgraph;

// Color settings
var CMap = {
  Red: {
    group: "UNUSED",
    bgcolor: "#d22",
    bordercolor: "#c11",
    linecolor: "#e22",
    arrowcolor: "#c22",
    arrowshape: "triangle",
    fromnode: "", // Assigned at startup
  },
  Orange: {
    group: "UNUSED",
    bgcolor: "#e90",
    bordercolor: "#d80",
    linecolor: "#fa0",
    arrowcolor: "#d80",
    arrowshape: "chevron",
    fromnode: "", // Assigned at startup
  },
  Yellow: {
    group: "UNUSED",
    bgcolor: "#fe0",
    bordercolor: "#dc0",
    linecolor: "#fe0",
    arrowcolor: "#cb0",
    arrowshape: "triangle",
    fromnode: "", // Assigned at startup
  },
  Lime: {
    group: "UNUSED",
    bgcolor: "#ac1",
    bordercolor: "#9b0",
    linecolor: "#ad1",
    arrowcolor: "#ab1",
    arrowshape: "chevron",
    fromnode: "", // Assigned at startup
  },
  Green: {
    group: "UNUSED",
    bgcolor: "#4b4",
    bordercolor: "#3a3",
    linecolor: "#4c4",
    arrowcolor: "#4a4",
    arrowshape: "triangle",
    fromnode: "", // Assigned at startup
  },
  Cyan: {
    group: "UNUSED",
    bgcolor: "#0cd",
    bordercolor: "#0bc",
    linecolor: "#0cc",
    arrowcolor: "#0aa",
    arrowshape: "chevron",
    fromnode: "", // Assigned at startup
  },
  Blue: {
    group: "UNUSED",
    bgcolor: "#44e",
    bordercolor: "#33d",
    linecolor: "#44f",
    arrowcolor: "#44d",
    arrowshape: "triangle",
    fromnode: "", // Assigned at startup
  },
  Violet: {
    group: "UNUSED",
    bgcolor: "#92c",
    bordercolor: "#81b",
    linecolor: "#92d",
    arrowcolor: "#92b",
    arrowshape: "chevron",
    fromnode: "", // Assigned at startup
  },
  Magenta: {
    group: "UNUSED",
    bgcolor: "#c0c",
    bordercolor: "#b0b",
    linecolor: "#d0d",
    arrowcolor: "#b0b",
    arrowshape: "triangle",
    fromnode: "", // Assigned at startup
  },
  Brown: {
    group: "UNUSED",
    bgcolor: "#b63",
    bordercolor: "#a53",
    linecolor: "#c73",
    arrowcolor: "#a53",
    arrowshape: "chevron",
    fromnode: "", // Assigned at startup
  },
};

var options = {
  animate: true, // whether to show the layout as it's running
  refresh: 1, // number of ticks per frame; higher is faster but more jerky
  maxSimulationTime: 3000, // max length in ms to run the layout
  ungrabifyWhileSimulating: false, // so you can't drag nodes during layout
  fit: true, // on every layout reposition of nodes, fit the viewport
  padding: 50, // padding around the simulation
  boundingBox: undefined, // constrain layout bounds; { x1, y1, x2, y2 } or { x1, y1, w, h }
  nodeDimensionsIncludeLabels: false, // whether labels should be included in determining the space used by a node

  // layout event callbacks
  ready: function () {}, // on layoutready
  stop: function () {}, // on layoutstop

  // positioning options
  randomize: false, // use random node positions at beginning of layout
  avoidOverlap: false, // if true, prevents overlap of node bounding boxes
  handleDisconnected: true, // if true, avoids disconnected components from overlapping
  convergenceThreshold: 0.01, // when the alpha value (system energy) falls below this value, the layout stops
  nodeSpacing: function (node) {
    return 10;
  }, // extra spacing around nodes
  flow: undefined, // use DAG/tree flow layout if specified, e.g. { axis: 'y', minSeparation: 30 }
  alignment: undefined, // relative alignment constraints on nodes, e.g. {vertical: [[{node: node1, offset: 0}, {node: node2, offset: 5}]], horizontal: [[{node: node3}, {node: node4}], [{node: node5}, {node: node6}]]}
  gapInequalities: undefined, // list of inequality constraints for the gap between the nodes, e.g. [{"axis":"y", "left":node1, "right":node2, "gap":25}]
  centerGraph: true, // adjusts the node positions initially to center the graph (pass false if you want to start the layout from the current position)

  // different methods of specifying edge length
  // each can be a constant numerical value or a function like `function( edge ){ return 2; }`
  edgeLength: undefined, // sets edge length directly in simulation
  edgeSymDiffLength: undefined, // symmetric diff edge length in simulation
  edgeJaccardLength: undefined, // jaccard edge length in simulation

  // iterations of cola algorithm; uses default values on undefined
  unconstrIter: undefined, // unconstrained initial layout iterations
  userConstIter: undefined, // initial layout iterations with user-specified constraints
  allConstIter: undefined, // initial layout iterations with all constraints including non-overlap
};

function AddNewPage(Player, ToString) {
  var ColorArray = [
    "Red",
    "Blue",
    "Green",
    "Yellow",
    "Cyan",
    "Orange",
    "Violet",
    "Magenta",
    "Lime",
    "Brown",
  ];

  for (let color of ColorArray) {
    if (CMap[color].group == Player) {
      AddNewElement(color, ToString);
      break;
    } else if (CMap[color].group == "UNUSED") {
      CMap[color].group = Player;
      AddNewElement(color, ToString);
      break;
    }
  }

  // The server calls this function to add new pages
}

function AddNewElement(PColor, ToString) {
  // Add a new edge and possibly a new node for a player click
  var CList = CMap[PColor];

  if (CList.fromnode == ToString) {
    return;
  }

  // Add a new node if it does not already exist
  if (!webgraph.getElementById(ToString).inside()) {
    webgraph.add({
      data: { id: ToString, group: CList.group },
      position: {
        x: webgraph.getElementById(CList.fromnode).position("x"),
        y: webgraph.getElementById(CList.fromnode).position("y"),
      },
    });
    webgraph
      .nodes('[group = "' + CList.group + '"]')
      .style("background-color", CList.bgcolor);
    webgraph
      .nodes('[group = "' + CList.group + '"]')
      .style("border-color", CList.bordercolor);
  }

  // Always add a new edge
  webgraph.add({
    data: {
      group: CList.group,
      source: CList.fromnode,
      target: ToString,
    },
  });
  webgraph
    .edges('[group = "' + CList.group + '"]')
    .style("line-color", CList.linecolor);
  webgraph
    .edges('[group = "' + CList.group + '"]')
    .style("target-arrow-color", CList.arrowcolor);
  webgraph
    .edges('[group = "' + CList.group + '"]')
    .style("target-arrow-shape", CList.arrowshape);

  // Reposition the player to the new node
  CList.fromnode = ToString;

  // Force a new layout
  var layout = webgraph.layout({ name: "cola", ...options });
  layout.run();
}

function StartGame(StartNode, GoalNode) {
  webgraph = cytoscape({
    container: document.getElementById("maincanvas"), // container to render in
    style: [
      // the stylesheet for the graph
      {
        selector: "node",
        style: {
          "background-color": "#fff",
          label: "data(id)",
          "font-size": 18,
          "text-outline-color": "#555",
          "text-outline-width": 1.6,
          color: "#fff",
          "border-width": 3,
          "border-color": "#bbb",
        },
      },
      {
        selector: "edge",
        style: {
          width: 4,
          //label: "data(group)", //Implement this as colorblind mode as a toggle
          "text-rotation": "autorotate",
          color: "#fff",
          "font-size": 10,
          "text-outline-color": "#000",
          "text-outline-width": 0.6,
          "target-arrow-shape": "triangle",
          "curve-style": "bezier",
        },
      },
    ],
  });

  webgraph.add({
    data: { id: StartNode, group: "Start" },
    position: { x: 0, y: 0 },
  });
  webgraph.add({
    data: { id: GoalNode, group: "Goal" },
    position: { x: 500, y: 0 },
  });

  for (let color in CMap) {
    CMap[color].fromnode = StartNode;
  }

  webgraph.nodes('[group = "Start"]').style("shape", "round-rectangle");
  webgraph.nodes('[group = "Start"]').style("text-outline-color", "#000");

  webgraph.nodes('[group = "Goal"]').style("shape", "star");
  webgraph.nodes('[group = "Goal"]').style("text-outline-color", "#000");

  var layout = webgraph.layout({ name: "cola", ...options });
  layout.run();
}

function createExampleGraph() {
  StartGame("That guy in the coca cola commercials", "Fish");

  // A three player example of a race between Santa Claus and Fish
  AddNewPage("Bob", "East-West Schism");
  AddNewPage("Bob", "Lent");
  AddNewPage("Bob", "Lent");
  AddNewPage("Bob", "Lent");
  AddNewPage("Bob", "Lent");
  AddNewPage("Bob", "Lent");
  AddNewPage("Bob", "Fish");

  AddNewPage("Charlie", "East-West Schism");
  AddNewPage("Charlie", "Lent");
  AddNewPage("Charlie", "Winter");

  AddNewPage("Mark", "Saint Nick");
  AddNewPage("Mark", "Christianty");
  AddNewPage("Mark", "Catholicism");
  AddNewPage("Mark", "Lent");
  AddNewPage("Mark", "Fish");

  AddNewPage("Alice", "Coca Cola");
  AddNewPage("Alice", "Atlanta");
  AddNewPage("Alice", "Georgia");

  AddNewPage("Emma", "East-West Schism");
  AddNewPage("Emma", "Passover");
  AddNewPage("Emma", "Pike");
  AddNewPage("Emma", "Passover");
  AddNewPage("Emma", "Carp");
  AddNewPage("Emma", "Rough Fish");
  AddNewPage("Emma", "Fish");

  AddNewPage("Robert", "Coca Cola");
  AddNewPage("Robert", "United States");
  AddNewPage("Robert", "Fish");

  AddNewPage("XXANTSLAYERXX", "Coca Cola");
  AddNewPage("XXANTSLAYERXX", "Pepsi Cola");
  AddNewPage("XXANTSLAYERXX", "Pepsi");

  AddNewPage("Your dad", "Coca Cola");
  AddNewPage("Your dad", "Pepsi Cola");
  AddNewPage("Your dad", "Pepsi");
  AddNewPage("Your dad", "Soda");
  AddNewPage("Your dad", "United States");

  AddNewPage("asdfghjk", "Beard");
  AddNewPage("asdfghjk", "Hair");
  AddNewPage("asdfghjk", "Head");

  AddNewPage("Paul", "East-West Schism");
  AddNewPage("Paul", "Passover");
  AddNewPage("Paul", "Carp");
  AddNewPage("Paul", "Aquaculture");
  AddNewPage("Paul", "Goldfish");
  AddNewPage("Paul", "Fish");
}

function CreateNicerExample() {
  StartGame("Santa Claus", "Fish");

  // A three player example of a race between Santa Claus and Fish
  AddNewPage("Bob", "East-West Schism");
  AddNewPage("Bob", "Lent");
  AddNewPage("Bob", "Fish");

  AddNewPage("Mark", "Saint Nick");
  AddNewPage("Mark", "Christianity");
  AddNewPage("Mark", "Catholicism");
  AddNewPage("Mark", "Lent");
  AddNewPage("Mark", "Fish");

  AddNewPage("Emma", "East-West Schism");
  AddNewPage("Emma", "Passover");
  AddNewPage("Emma", "Pike");
  AddNewPage("Emma", "Passover");
  AddNewPage("Emma", "Carp");
  AddNewPage("Emma", "Rough Fish");
  AddNewPage("Emma", "Fish");

  AddNewPage("Robert", "Pepsi");
  AddNewPage("Robert", "Fat");
  AddNewPage("Robert", "Tuna");
  AddNewPage("Robert", "Game Fish");
  AddNewPage("Robert", "Fish");

  AddNewPage("XXANTSLAYERXX", "Coca Cola");
  AddNewPage("XXANTSLAYERXX", "Pepsi Cola");
  AddNewPage("XXANTSLAYERXX", "Pepsi");

  AddNewPage("Paul", "East-West Schism");
  AddNewPage("Paul", "Passover");
  AddNewPage("Paul", "Carp");
  AddNewPage("Paul", "Aquaculture");
  AddNewPage("Paul", "Goldfish");
  AddNewPage("Paul", "Fish");
}
