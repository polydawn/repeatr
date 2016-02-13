Why Repeat?
===========

Repeatability is the cornerstone of science and engineering.

`repeatr` is about making it easy for all your digital stuff.



What is this good for?
----------------------

Data processing.  Science.  Building software.  Doing analysis.  Responsible journalism.
Anywhere where it's appropriate to show your work, it's appropriate to use Repeatr.

If you're a programmer, think of it like source control, but for your entire environment, with precise commits, rollback, even bisect capabilities -- and it's essential to how you share your work with others.

If you're a researcher, think of it like your lab notebook, but you write the notebook first, and then Repeatr runs the experiment *for* you according to your exact instructions.

If you're a journalist, think of it like citing sources, but not only can readers and editors look up your citations, they can also run the analysis themselves in one click.

In any situation where quality is critical and transparency is a must, Repeatr can help you raise the bar.

### As DevOps

- Sandboxing to make sure your processes are consistent -- like a Continuous Integration system, but also ready to help you debug consistency itself.
- Immutable deployments -- and we mean it.  Once the formula is committed, everything is pinned down and guaranteed to deliver.  Ops teams delivering IT infrastructure can rest easy.
- Repeatable, reproducible pipelines -- when transparency is important, Repeatr makes planning work and leaving an audit log one and the same: exact reproducibility is simply natural.
- Roll it back -- formulas used in the past continue to work.  Forever.  If you change your data, and later discover you want to run with an older configuration again, that's *always* possible.

### As Data Science

- Data warehousing -- where file corruption can mean millions of dollars in damage from inaccurate data or expensive calculations that need to be re-run, Repeatr detects issues before they cause problems.  Storing data on untrusted third-party storage is safe because integrity guarantees are baked in to the system.
- Checking Your Work -- reproducible computation by default means it's much easier to ferret out inconsistencies in data or imprecisions in analysis.
- Sharing your Analysis -- after you write a Formula with your analysis, share it with anyone in your team (or your publishing journal's review board!).  The Formula is self-contained -- no setup, no nonsense; once you hand off that Formula, you can sign off knowing everyone else can keep running exactly where you left off.

### As Trusted Computing

Today more than ever, computing is central to our lives.
We need trust in our digital systems to be safe and secure:
in everything from communicating with friends to running businesses confidently,
to keeping power plants running.  For some of us, it might even be as close to
home as software keeping a mechanical heart beating.
Confidence is our computers is not optional; it's a survival requirement.

And yet, even within Open Source software, how do we really trust that programs
we run are the ones we think we have the source to?

We don't.

This is a known problem.
We've had [CVEs where a single bit flipped](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2002-0083)
in the executable was dangerous.
And
[History](https://twitter.com/bcrypt/status/645757802384719872)
[Repeats](https://theintercept.com/2015/03/10/ispy-cia-campaign-steal-apples-secrets/)
[Itself](https://people.torproject.org/~mikeperry/transient/2014MozillaReproducible.pdf).

Open Source is great.  Package signing is great.  *_Neither_ _is_ _enough_!*
We need reproducible builds so we can make a link between the source and
binary that anyone can audit, and anyone can reproduce.

(Repeatr Formulas)[formulas.md] describe that link.  They are, by design,
  - a concise
    - language-agonostic
      - distro-agnostic
        - well-defined canonical format
... which precisely captures an environment and process so we can permanently
capture the transitions from source to binary.

Package a Formula with your binary when you release software; sign the whole thing.

Let's go build some trusted software, eh?
